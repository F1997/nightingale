package memsto

import (
	"fmt"
	"sync"
	"time"

	"github.com/F1997/nightingale/dumper"
	"github.com/F1997/nightingale/models"
	"github.com/F1997/nightingale/pkg/ctx"

	"github.com/pkg/errors"
	"github.com/toolkits/pkg/logger"
)

type AlertMuteCacheType struct {
	statTotal       int64
	statLastUpdated int64
	ctx             *ctx.Context
	stats           *Stats

	sync.RWMutex
	mutes map[int64][]*models.AlertMute // key: busi_group_id
}

// 创建一个新的告警屏蔽数据缓存实例，并初始化相关数据
func NewAlertMuteCache(ctx *ctx.Context, stats *Stats) *AlertMuteCacheType {
	amc := &AlertMuteCacheType{
		statTotal:       -1,
		statLastUpdated: -1,
		ctx:             ctx,
		stats:           stats,
		mutes:           make(map[int64][]*models.AlertMute),
	}
	amc.SyncAlertMutes()
	return amc
}

// 重置缓存
func (amc *AlertMuteCacheType) Reset() {
	amc.Lock()
	defer amc.Unlock()

	amc.statTotal = -1
	amc.statLastUpdated = -1
	amc.mutes = make(map[int64][]*models.AlertMute)
}

// 判断数据是否发生了变化，总记录数和最后更新时间发生了变化，返回 true，否则返回 false
func (amc *AlertMuteCacheType) StatChanged(total, lastUpdated int64) bool {
	if amc.statTotal == total && amc.statLastUpdated == lastUpdated {
		return false
	}

	return true
}

// 设置缓存数据，查询到的最新数据存储在缓存
func (amc *AlertMuteCacheType) Set(ms map[int64][]*models.AlertMute, total, lastUpdated int64) {
	amc.Lock()
	amc.mutes = ms
	amc.Unlock()

	// only one goroutine used, so no need lock
	amc.statTotal = total
	amc.statLastUpdated = lastUpdated
}

// 根据 bgid 获取数据列表
func (amc *AlertMuteCacheType) Gets(bgid int64) ([]*models.AlertMute, bool) {
	amc.RLock()
	defer amc.RUnlock()
	lst, has := amc.mutes[bgid]
	return lst, has
}

// 获取所有告警屏蔽数据
func (amc *AlertMuteCacheType) GetAllStructs() map[int64][]models.AlertMute {
	amc.RLock()
	defer amc.RUnlock()

	ret := make(map[int64][]models.AlertMute)
	for bgid := range amc.mutes {
		lst := amc.mutes[bgid]
		for i := 0; i < len(lst); i++ {
			ret[bgid] = append(ret[bgid], *lst[i])
		}
	}

	return ret
}

// 同步告警屏蔽数据
func (amc *AlertMuteCacheType) SyncAlertMutes() {
	err := amc.syncAlertMutes()
	if err != nil {
		fmt.Println("failed to sync alert mutes:", err)
		exit(1)
	}

	go amc.loopSyncAlertMutes()
}

// 定时同步数据
func (amc *AlertMuteCacheType) loopSyncAlertMutes() {
	duration := time.Duration(9000) * time.Millisecond
	for {
		time.Sleep(duration)
		if err := amc.syncAlertMutes(); err != nil {
			logger.Warning("failed to sync alert mutes:", err)
		}
	}
}

// 执行告警屏蔽数据的同步操作
func (amc *AlertMuteCacheType) syncAlertMutes() error {
	// 记录开始时间
	start := time.Now()

	// 查询统计信息
	stat, err := models.AlertMuteStatistics(amc.ctx)
	if err != nil {
		// 将错误信息写入日志
		dumper.PutSyncRecord("alert_mutes", start.Unix(), -1, -1, "failed to query statistics: "+err.Error())
		return errors.WithMessage(err, "failed to exec AlertMuteStatistics")
	}

	// 判断数据是否发生变化，如果没变，将 Prometheus 指标设置为0，将同步结果写入日志，并返回。
	if !amc.StatChanged(stat.Total, stat.LastUpdated) {
		amc.stats.GaugeCronDuration.WithLabelValues("sync_alert_mutes").Set(0)
		amc.stats.GaugeSyncNumber.WithLabelValues("sync_alert_mutes").Set(0)
		dumper.PutSyncRecord("alert_mutes", start.Unix(), -1, -1, "not changed")
		return nil
	}

	// 查询所有告警屏蔽数据
	lst, err := models.AlertMuteGetsAll(amc.ctx)
	if err != nil {
		// 错误写入日志
		dumper.PutSyncRecord("alert_mutes", start.Unix(), -1, -1, "failed to query records: "+err.Error())
		return errors.WithMessage(err, "failed to exec AlertMuteGetsByCluster")
	}
	// 遍历查询到的数据，存入 oks 中
	oks := make(map[int64][]*models.AlertMute)
	for i := 0; i < len(lst); i++ {
		err = lst[i].Parse()
		if err != nil {
			logger.Warningf("failed to parse alert_mute, id: %d", lst[i].Id)
			continue
		}

		oks[lst[i].GroupId] = append(oks[lst[i].GroupId], lst[i])
	}

	// 更新缓存数据，将 oks 数据更新到缓存
	amc.Set(oks, stat.Total, stat.LastUpdated)

	// 计算耗时，记录同步数据信息
	ms := time.Since(start).Milliseconds()
	amc.stats.GaugeCronDuration.WithLabelValues("sync_alert_mutes").Set(float64(ms))
	amc.stats.GaugeSyncNumber.WithLabelValues("sync_alert_mutes").Set(float64(len(lst)))
	logger.Infof("timer: sync mutes done, cost: %dms, number: %d", ms, len(lst))
	dumper.PutSyncRecord("alert_mutes", start.Unix(), ms, len(lst), "success")

	return nil
}

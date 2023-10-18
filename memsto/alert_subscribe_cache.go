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

type AlertSubscribeCacheType struct {
	statTotal       int64
	statLastUpdated int64
	ctx             *ctx.Context
	stats           *Stats

	sync.RWMutex
	subs map[int64][]*models.AlertSubscribe
}

// 初始化 AlertSubscribeCacheType
func NewAlertSubscribeCache(ctx *ctx.Context, stats *Stats) *AlertSubscribeCacheType {
	asc := &AlertSubscribeCacheType{
		statTotal:       -1,
		statLastUpdated: -1,
		ctx:             ctx,
		stats:           stats,
		subs:            make(map[int64][]*models.AlertSubscribe),
	}
	asc.SyncAlertSubscribes()
	return asc
}

// 重置缓存，清除之前的数据
func (c *AlertSubscribeCacheType) Reset() {
	c.Lock()
	defer c.Unlock()

	c.statTotal = -1
	c.statLastUpdated = -1
	c.subs = make(map[int64][]*models.AlertSubscribe)
}

// 判断统计数据是否发生变化
func (c *AlertSubscribeCacheType) StatChanged(total, lastUpdated int64) bool {
	if c.statTotal == total && c.statLastUpdated == lastUpdated {
		return false
	}

	return true
}

func (c *AlertSubscribeCacheType) Set(m map[int64][]*models.AlertSubscribe, total, lastUpdated int64) {
	c.Lock()
	c.subs = m
	c.Unlock()

	// only one goroutine used, so no need lock
	c.statTotal = total
	c.statLastUpdated = lastUpdated
}

func (c *AlertSubscribeCacheType) Get(ruleId int64) ([]*models.AlertSubscribe, bool) {
	c.RLock()
	defer c.RUnlock()

	lst, has := c.subs[ruleId]
	return lst, has
}

func (c *AlertSubscribeCacheType) GetAll() []*models.AlertSubscribe {
	c.RLock()
	defer c.RUnlock()
	var ret []*models.AlertSubscribe
	for _, v := range c.subs {
		ret = append(ret, v...)
	}
	return ret
}

func (c *AlertSubscribeCacheType) GetStructs(ruleId int64) []models.AlertSubscribe {
	c.RLock()
	defer c.RUnlock()

	lst, has := c.subs[ruleId]
	if !has {
		return []models.AlertSubscribe{}
	}

	ret := make([]models.AlertSubscribe, len(lst))
	for i := 0; i < len(lst); i++ {
		ret[i] = *lst[i]
	}

	return ret
}

func (c *AlertSubscribeCacheType) SyncAlertSubscribes() {
	err := c.syncAlertSubscribes()
	if err != nil {
		fmt.Println("failed to sync alert subscribes:", err)
		exit(1)
	}

	go c.loopSyncAlertSubscribes()
}

// 定期执行同步操作
func (c *AlertSubscribeCacheType) loopSyncAlertSubscribes() {
	duration := time.Duration(9000) * time.Millisecond
	for {
		time.Sleep(duration)
		if err := c.syncAlertSubscribes(); err != nil {
			logger.Warning("failed to sync alert subscribes:", err)
		}
	}
}

// 执行同步操作
func (c *AlertSubscribeCacheType) syncAlertSubscribes() error {
	// 记录开始时间
	start := time.Now()
	// 查询数据
	stat, err := models.AlertSubscribeStatistics(c.ctx)
	if err != nil {
		dumper.PutSyncRecord("alert_subscribes", start.Unix(), -1, -1, "failed to query statistics: "+err.Error())
		return errors.WithMessage(err, "failed to exec AlertSubscribeStatistics")
	}
	// 数据是否变化
	if !c.StatChanged(stat.Total, stat.LastUpdated) {
		c.stats.GaugeCronDuration.WithLabelValues("sync_alert_subscribes").Set(0)
		c.stats.GaugeSyncNumber.WithLabelValues("sync_alert_subscribes").Set(0)
		dumper.PutSyncRecord("alert_subscribes", start.Unix(), -1, -1, "not changed")
		return nil
	}

	// 获取所有告警订阅规则数据
	lst, err := models.AlertSubscribeGetsAll(c.ctx)
	if err != nil {
		dumper.PutSyncRecord("alert_subscribes", start.Unix(), -1, -1, "failed to query records: "+err.Error())
		return errors.WithMessage(err, "failed to exec AlertSubscribeGetsAll")
	}

	// 创建空的 subs map，用于存储告警订阅信息，键是规则的 ID，值是订阅信息的切片
	subs := make(map[int64][]*models.AlertSubscribe)

	// 遍历查询到的告警订阅数据
	for i := 0; i < len(lst); i++ {
		// 对订阅数据进行解析
		err = lst[i].Parse()
		if err != nil {
			logger.Warningf("failed to parse alert subscribe, id: %d", lst[i].Id)
			continue
		}
		// 将数据库字段映射到前端字段
		err = lst[i].DB2FE()
		if err != nil {
			logger.Warningf("failed to db2fe alert subscribe, id: %d", lst[i].Id)
			continue
		}
		// 填充数据源的 ID 信息，以确保订阅信息包含了所需的数据源信息
		err = lst[i].FillDatasourceIds(c.ctx)
		if err != nil {
			logger.Warningf("failed to fill datasource ids, id: %d", lst[i].Id)
			continue
		}
		// 订阅信息添加到 subs map 中，根据规则的 ID 进行分类
		subs[lst[i].RuleId] = append(subs[lst[i].RuleId], lst[i])
	}
	// 更新数据&性能指标数据到 AlertSubscribeCacheType 实例
	c.Set(subs, stat.Total, stat.LastUpdated)

	// 记录同步耗时和同步的数据数量，并将同步记录保存到日志中
	ms := time.Since(start).Milliseconds()
	c.stats.GaugeCronDuration.WithLabelValues("sync_alert_subscribes").Set(float64(ms))
	c.stats.GaugeSyncNumber.WithLabelValues("sync_alert_subscribes").Set(float64(len(lst)))
	logger.Infof("timer: sync subscribes done, cost: %dms, number: %d", ms, len(lst))
	dumper.PutSyncRecord("alert_subscribes", start.Unix(), ms, len(lst), "success")

	return nil
}

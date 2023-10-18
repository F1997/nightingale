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

type AlertRuleCacheType struct {
	statTotal       int64
	statLastUpdated int64
	ctx             *ctx.Context
	stats           *Stats

	sync.RWMutex
	rules map[int64]*models.AlertRule // key: rule id
}

func NewAlertRuleCache(ctx *ctx.Context, stats *Stats) *AlertRuleCacheType {
	arc := &AlertRuleCacheType{
		statTotal:       -1,
		statLastUpdated: -1,
		ctx:             ctx,
		stats:           stats,
		rules:           make(map[int64]*models.AlertRule),
	}
	arc.SyncAlertRules()
	return arc
}

func (arc *AlertRuleCacheType) Reset() {
	arc.Lock()
	defer arc.Unlock()

	arc.statTotal = -1
	arc.statLastUpdated = -1
	arc.rules = make(map[int64]*models.AlertRule)
}

func (arc *AlertRuleCacheType) StatChanged(total, lastUpdated int64) bool {
	if arc.statTotal == total && arc.statLastUpdated == lastUpdated {
		return false
	}

	return true
}

func (arc *AlertRuleCacheType) Set(m map[int64]*models.AlertRule, total, lastUpdated int64) {
	arc.Lock()
	arc.rules = m
	arc.Unlock()

	// only one goroutine used, so no need lock
	arc.statTotal = total
	arc.statLastUpdated = lastUpdated
}

func (arc *AlertRuleCacheType) Get(ruleId int64) *models.AlertRule {
	arc.RLock()
	defer arc.RUnlock()
	return arc.rules[ruleId]
}

func (arc *AlertRuleCacheType) GetRuleIds() []int64 {
	arc.RLock()
	defer arc.RUnlock()

	count := len(arc.rules)
	list := make([]int64, 0, count)
	for ruleId := range arc.rules {
		list = append(list, ruleId)
	}

	return list
}

func (arc *AlertRuleCacheType) SyncAlertRules() {
	err := arc.syncAlertRules()
	if err != nil {
		fmt.Println("failed to sync alert rules:", err)
		exit(1)
	}

	go arc.loopSyncAlertRules()
}

func (arc *AlertRuleCacheType) loopSyncAlertRules() {
	duration := time.Duration(9000) * time.Millisecond
	for {
		time.Sleep(duration)
		if err := arc.syncAlertRules(); err != nil {
			logger.Warning("failed to sync alert rules:", err)
		}
	}
}

// 同步告警规则到缓存
func (arc *AlertRuleCacheType) syncAlertRules() error {
	// 记录开始时间
	start := time.Now()
	// 查询告警规则统计信息
	stat, err := models.AlertRuleStatistics(arc.ctx)
	if err != nil {
		dumper.PutSyncRecord("alert_rules", start.Unix(), -1, -1, "failed to query statistics: "+err.Error())
		return errors.WithMessage(err, "failed to exec AlertRuleStatistics")
	}
	// 判断数据是否发生变化
	if !arc.StatChanged(stat.Total, stat.LastUpdated) {
		arc.stats.GaugeCronDuration.WithLabelValues("sync_alert_rules").Set(0)
		arc.stats.GaugeSyncNumber.WithLabelValues("sync_alert_rules").Set(0)
		dumper.PutSyncRecord("alert_rules", start.Unix(), -1, -1, "not changed")
		return nil
	}

	// 查询所有告警规则数据
	lst, err := models.AlertRuleGetsAll(arc.ctx)
	if err != nil {
		dumper.PutSyncRecord("alert_rules", start.Unix(), -1, -1, "failed to query records: "+err.Error())
		return errors.WithMessage(err, "failed to exec AlertRuleGetsByCluster")
	}

	// 遍历查询到的警报规则数据，存入 m, key 是规则的 ID，value 是对应的告警规则
	m := make(map[int64]*models.AlertRule)
	for i := 0; i < len(lst); i++ {
		m[lst[i].Id] = lst[i]
	}

	// 更新缓存数据
	arc.Set(m, stat.Total, stat.LastUpdated)
	// 计算同步耗时，记录同步完成信息
	ms := time.Since(start).Milliseconds()
	arc.stats.GaugeCronDuration.WithLabelValues("sync_alert_rules").Set(float64(ms))
	arc.stats.GaugeSyncNumber.WithLabelValues("sync_alert_rules").Set(float64(len(m)))
	logger.Infof("timer: sync rules done, cost: %dms, number: %d", ms, len(m))
	dumper.PutSyncRecord("alert_rules", start.Unix(), ms, len(m), "success")

	return nil
}

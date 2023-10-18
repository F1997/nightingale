package alert

import (
	"context"
	"fmt"

	"github.com/F1997/nightingale/alert/aconf"
	"github.com/F1997/nightingale/alert/astats"
	"github.com/F1997/nightingale/alert/dispatch"
	"github.com/F1997/nightingale/alert/eval"
	"github.com/F1997/nightingale/alert/naming"
	"github.com/F1997/nightingale/alert/process"
	"github.com/F1997/nightingale/alert/queue"
	"github.com/F1997/nightingale/alert/record"
	"github.com/F1997/nightingale/alert/router"
	"github.com/F1997/nightingale/alert/sender"
	"github.com/F1997/nightingale/conf"
	"github.com/F1997/nightingale/dumper"
	"github.com/F1997/nightingale/memsto"
	"github.com/F1997/nightingale/models"
	"github.com/F1997/nightingale/pkg/ctx"
	"github.com/F1997/nightingale/pkg/httpx"
	"github.com/F1997/nightingale/pkg/logx"
	"github.com/F1997/nightingale/prom"
	"github.com/F1997/nightingale/pushgw/pconf"
	"github.com/F1997/nightingale/pushgw/writer"
)

func Initialize(configDir string, cryptoKey string) (func(), error) {
	config, err := conf.InitConfig(configDir, cryptoKey)
	if err != nil {
		return nil, fmt.Errorf("failed to init config: %v", err)
	}

	logxClean, err := logx.Init(config.Log)
	if err != nil {
		return nil, err
	}

	ctx := ctx.NewContext(context.Background(), nil, false, config.CenterApi)

	syncStats := memsto.NewSyncStats()
	alertStats := astats.NewSyncStats()

	targetCache := memsto.NewTargetCache(ctx, syncStats, nil)
	busiGroupCache := memsto.NewBusiGroupCache(ctx, syncStats)
	alertMuteCache := memsto.NewAlertMuteCache(ctx, syncStats)
	alertRuleCache := memsto.NewAlertRuleCache(ctx, syncStats)
	notifyConfigCache := memsto.NewNotifyConfigCache(ctx)
	dsCache := memsto.NewDatasourceCache(ctx, syncStats)
	userCache := memsto.NewUserCache(ctx, syncStats)
	userGroupCache := memsto.NewUserGroupCache(ctx, syncStats)

	promClients := prom.NewPromClient(ctx, config.Alert.Heartbeat)

	externalProcessors := process.NewExternalProcessors()

	Start(config.Alert, config.Pushgw, syncStats, alertStats, externalProcessors, targetCache, busiGroupCache, alertMuteCache, alertRuleCache, notifyConfigCache, dsCache, ctx, promClients, userCache, userGroupCache)

	r := httpx.GinEngine(config.Global.RunMode, config.HTTP)
	rt := router.New(config.HTTP, config.Alert, alertMuteCache, targetCache, busiGroupCache, alertStats, ctx, externalProcessors)
	rt.Config(r)
	dumper.ConfigRouter(r)

	httpClean := httpx.Init(config.HTTP, r)

	return func() {
		logxClean()
		httpClean()
	}, nil
}

func Start(alertc aconf.Alert, pushgwc pconf.Pushgw, syncStats *memsto.Stats, alertStats *astats.Stats, externalProcessors *process.ExternalProcessorsType, targetCache *memsto.TargetCacheType, busiGroupCache *memsto.BusiGroupCacheType,
	alertMuteCache *memsto.AlertMuteCacheType, alertRuleCache *memsto.AlertRuleCacheType, notifyConfigCache *memsto.NotifyConfigCacheType, datasourceCache *memsto.DatasourceCacheType, ctx *ctx.Context, promClients *prom.PromClientMap, userCache *memsto.UserCacheType, userGroupCache *memsto.UserGroupCacheType) {
	alertSubscribeCache := memsto.NewAlertSubscribeCache(ctx, syncStats)
	recordingRuleCache := memsto.NewRecordingRuleCache(ctx, syncStats)

	go models.InitNotifyConfig(ctx, alertc.Alerting.TemplatesDir)

	naming := naming.NewNaming(ctx, alertc.Heartbeat)

	writers := writer.NewWriters(pushgwc)
	record.NewScheduler(alertc, recordingRuleCache, promClients, writers, alertStats)

	eval.NewScheduler(alertc, externalProcessors, alertRuleCache, targetCache, busiGroupCache, alertMuteCache, datasourceCache, promClients, naming, ctx, alertStats)

	dp := dispatch.NewDispatch(alertRuleCache, userCache, userGroupCache, alertSubscribeCache, targetCache, notifyConfigCache, alertc.Alerting, ctx)
	consumer := dispatch.NewConsumer(alertc.Alerting, ctx, dp)

	go dp.ReloadTpls()
	go consumer.LoopConsume()

	go queue.ReportQueueSize(alertStats)
	go sender.InitEmailSender(notifyConfigCache.GetSMTP())
}

package center

import (
	"context"
	"fmt"

	"github.com/F1997/nightingale/center/cconf"
	"github.com/F1997/nightingale/center/cstats"
	"github.com/F1997/nightingale/center/metas"
	"github.com/F1997/nightingale/center/sso"
	"github.com/F1997/nightingale/conf"
	"github.com/F1997/nightingale/dumper"
	"github.com/F1997/nightingale/memsto"
	"github.com/F1997/nightingale/models"
	"github.com/F1997/nightingale/models/migrate"
	"github.com/F1997/nightingale/pkg/ctx"
	"github.com/F1997/nightingale/pkg/httpx"
	"github.com/F1997/nightingale/pkg/i18nx"
	"github.com/F1997/nightingale/pkg/logx"
	"github.com/F1997/nightingale/pkg/version"
	"github.com/F1997/nightingale/prom"
	"github.com/F1997/nightingale/pushgw/idents"
	"github.com/F1997/nightingale/pushgw/writer"
	"github.com/F1997/nightingale/storage"

	// alertrt "github.com/F1997/nightingale/alert/router"
	centerrt "github.com/F1997/nightingale/center/router"
	pushgwrt "github.com/F1997/nightingale/pushgw/router"
)

func Initialize(configDir string, cryptoKey string) (func(), error) {
	// 初始化应用程序的配置
	config, err := conf.InitConfig(configDir, cryptoKey)
	if err != nil {
		return nil, fmt.Errorf("failed to init config: %v", err)
	}

	cconf.LoadMetricsYaml(configDir, config.Center.MetricsYamlFile)
	cconf.LoadOpsYaml(configDir, config.Center.OpsYamlFile)

	// 初始化日志记录器， 用于记录应用程序的日志信息
	logxClean, err := logx.Init(config.Log)
	if err != nil {
		return nil, err
	}

	// 初始化 I18N，用于多语言支持
	i18nx.Init(configDir)
	// 初始化统计信息模块，用于记录应用程序的统计信息。
	cstats.Init()

	// 初始化数据库连接
	db, err := storage.New(config.DB)
	if err != nil {
		return nil, err
	}
	// 使用数据库创建一个上下文
	ctx := ctx.NewContext(context.Background(), db, true)
	models.InitRoot(ctx)
	// 进行数据库迁移操作
	migrate.Migrate(db)

	// 初始化 Redis 连接
	redis, err := storage.NewRedis(config.Redis)
	if err != nil {
		return nil, err
	}

	metas := metas.New(redis)
	idents := idents.New(ctx)

	syncStats := memsto.NewSyncStats()
	// alertStats := astats.NewSyncStats()

	// 初始化单点登录
	sso := sso.Init(config.Center, ctx)

	// 创建缓存
	busiGroupCache := memsto.NewBusiGroupCache(ctx, syncStats)
	targetCache := memsto.NewTargetCache(ctx, syncStats, redis)
	dsCache := memsto.NewDatasourceCache(ctx, syncStats)
	// alertMuteCache := memsto.NewAlertMuteCache(ctx, syncStats)
	// alertRuleCache := memsto.NewAlertRuleCache(ctx, syncStats)
	notifyConfigCache := memsto.NewNotifyConfigCache(ctx)
	userCache := memsto.NewUserCache(ctx, syncStats)
	userGroupCache := memsto.NewUserGroupCache(ctx, syncStats)

	// 初始化 Prometheus 客户端
	promClients := prom.NewPromClient(ctx, config.Alert.Heartbeat)

	// // 初始化外部处理器，用于处理外部请求
	// externalProcessors := process.NewExternalProcessors()

	// // 启动告警系统
	// alert.Start(config.Alert, config.Pushgw, syncStats, alertStats, externalProcessors, targetCache, busiGroupCache, alertMuteCache, alertRuleCache, notifyConfigCache, dsCache, ctx, promClients, userCache, userGroupCache)

	// 初始化推送网关写入器
	writers := writer.NewWriters(config.Pushgw)

	// 初始化 RSA 配置
	httpx.InitRSAConfig(&config.HTTP.RSA)
	// 启动一个后台 goroutine 用于获取应用程序的 Github 版本信息
	go version.GetGithubVersion()

	// 告警路由，中心路由，推送网关路由
	// alertrtRouter := alertrt.New(config.HTTP, config.Alert, alertMuteCache, targetCache, busiGroupCache, alertStats, ctx, externalProcessors)
	centerRouter := centerrt.New(config.HTTP, config.Center, cconf.Operations, dsCache, notifyConfigCache, promClients, redis, sso, ctx, metas, idents, targetCache, userCache, userGroupCache)
	pushgwRouter := pushgwrt.New(config.HTTP, config.Pushgw, targetCache, busiGroupCache, idents, writers, ctx)

	// 创建路由引擎
	r := httpx.GinEngine(config.Global.RunMode, config.HTTP)

	centerRouter.Config(r)
	// alertrtRouter.Config(r)
	pushgwRouter.Config(r)
	dumper.ConfigRouter(r)

	httpClean := httpx.Init(config.HTTP, r)

	// 返回清理函数，用于应用程序退出时进行清理工作
	return func() {
		logxClean()
		httpClean()
	}, nil
}

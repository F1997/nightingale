package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/F1997/nightingale/alert"
	"github.com/F1997/nightingale/alert/astats"
	"github.com/F1997/nightingale/alert/process"
	"github.com/F1997/nightingale/conf"
	"github.com/F1997/nightingale/dumper"
	"github.com/F1997/nightingale/memsto"
	"github.com/F1997/nightingale/pkg/ctx"
	"github.com/F1997/nightingale/pkg/httpx"
	"github.com/F1997/nightingale/pkg/logx"
	"github.com/F1997/nightingale/prom"
	"github.com/F1997/nightingale/pushgw/idents"
	"github.com/F1997/nightingale/pushgw/writer"

	alertrt "github.com/F1997/nightingale/alert/router"
	pushgwrt "github.com/F1997/nightingale/pushgw/router"
)

// 初始化 Nightingale 的警报模块和推送网关模块，包括配置、日志、缓存、路由等，然后返回一个清理函数，用于在程序退出时执行清理操作
func Initialize(configDir string, cryptoKey string) (func(), error) {
	// 初始化配置
	config, err := conf.InitConfig(configDir, cryptoKey)
	if err != nil {
		return nil, fmt.Errorf("failed to init config: %v", err)
	}

	// 初始化日志记录器
	logxClean, err := logx.Init(config.Log)
	if err != nil {
		return nil, err
	}
	//check CenterApi is default value
	if len(config.CenterApi.Addrs) < 1 {
		return nil, errors.New("failed to init config: the CenterApi configuration is missing")
	}
	// 创建一个包含了 CenterApi 的配置信息的上下文
	ctx := ctx.NewContext(context.Background(), nil, false, config.CenterApi)

	// 创建同步统计信息的对象
	syncStats := memsto.NewSyncStats()

	// 创建缓存目标信息的对象
	targetCache := memsto.NewTargetCache(ctx, syncStats, nil)
	// 创建缓存业务分组信息的对象
	busiGroupCache := memsto.NewBusiGroupCache(ctx, syncStats)
	// 创建标识目标的对象
	idents := idents.New(ctx)
	// 创建用于写入推送网关的对象
	writers := writer.NewWriters(config.Pushgw)
	// 创建用于配置推送网关路由的对象
	pushgwRouter := pushgwrt.New(config.HTTP, config.Pushgw, targetCache, busiGroupCache, idents, writers, ctx)
	// 创建一个 Gin 引擎对象
	r := httpx.GinEngine(config.Global.RunMode, config.HTTP)
	// 配置推送网关的路由
	pushgwRouter.Config(r)

	// 告警模块没有被禁用，创建告警相关对象
	if !config.Alert.Disable {
		alertStats := astats.NewSyncStats()
		dsCache := memsto.NewDatasourceCache(ctx, syncStats)
		alertMuteCache := memsto.NewAlertMuteCache(ctx, syncStats)
		alertRuleCache := memsto.NewAlertRuleCache(ctx, syncStats)
		notifyConfigCache := memsto.NewNotifyConfigCache(ctx)
		userCache := memsto.NewUserCache(ctx, syncStats)
		userGroupCache := memsto.NewUserGroupCache(ctx, syncStats)

		promClients := prom.NewPromClient(ctx, config.Alert.Heartbeat)
		externalProcessors := process.NewExternalProcessors()
		// 启动告警模块
		alert.Start(config.Alert, config.Pushgw, syncStats, alertStats, externalProcessors, targetCache, busiGroupCache, alertMuteCache, alertRuleCache, notifyConfigCache, dsCache, ctx, promClients, userCache, userGroupCache)
		// 创建配置告警模块的路由对象
		alertrtRouter := alertrt.New(config.HTTP, config.Alert, alertMuteCache, targetCache, busiGroupCache, alertStats, ctx, externalProcessors)

		// 配置告警模块的路由
		alertrtRouter.Config(r)
	}
	// 配置路由
	dumper.ConfigRouter(r)
	// 初始化 HTTP 服务
	httpClean := httpx.Init(config.HTTP, r)

	// 返回一个清理函数
	return func() {
		logxClean()
		httpClean()
	}, nil
}

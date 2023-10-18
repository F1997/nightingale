package pushgw

import (
	"context"
	"fmt"

	"github.com/F1997/nightingale/conf"
	"github.com/F1997/nightingale/memsto"
	"github.com/F1997/nightingale/pkg/ctx"
	"github.com/F1997/nightingale/pkg/httpx"
	"github.com/F1997/nightingale/pkg/logx"
	"github.com/F1997/nightingale/pushgw/idents"
	"github.com/F1997/nightingale/pushgw/router"
	"github.com/F1997/nightingale/pushgw/writer"
)

type PushgwProvider struct {
	Ident  *idents.Set
	Router *router.Router
}

// 初始化 Pushgateway
func Initialize(configDir string, cryptoKey string) (func(), error) {
	// 初始化配置信息
	config, err := conf.InitConfig(configDir, cryptoKey)
	if err != nil {
		return nil, fmt.Errorf("failed to init config: %v", err)
	}
	// 初始化日志系统
	logxClean, err := logx.Init(config.Log)
	if err != nil {
		return nil, err
	}

	ctx := ctx.NewContext(context.Background(), nil, false, config.CenterApi)

	// 用于标识管理
	idents := idents.New(ctx)
	// 用于统计信息
	stats := memsto.NewSyncStats()

	// busiGroupCache 缓存业务分组 targetCache 缓存目标信息
	busiGroupCache := memsto.NewBusiGroupCache(ctx, stats)
	targetCache := memsto.NewTargetCache(ctx, stats, nil)

	// 实例化写入器
	writers := writer.NewWriters(config.Pushgw)

	// gin 引擎
	r := httpx.GinEngine(config.Global.RunMode, config.HTTP)
	// 实例化 路由处理器
	rt := router.New(config.HTTP, config.Pushgw, targetCache, busiGroupCache, idents, writers, ctx)
	// 装载路由处理器
	rt.Config(r)

	httpClean := httpx.Init(config.HTTP, r)

	return func() {
		logxClean()
		httpClean()
	}, nil
}

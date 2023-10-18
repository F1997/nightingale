package router

import (
	"fmt"
	"net/http"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/F1997/nightingale/center/cconf"
	"github.com/F1997/nightingale/center/cstats"
	"github.com/F1997/nightingale/center/metas"
	"github.com/F1997/nightingale/center/sso"
	_ "github.com/F1997/nightingale/front/statik"
	"github.com/F1997/nightingale/memsto"
	"github.com/F1997/nightingale/pkg/aop"
	"github.com/F1997/nightingale/pkg/ctx"
	"github.com/F1997/nightingale/pkg/httpx"
	"github.com/F1997/nightingale/pkg/version"
	"github.com/F1997/nightingale/prom"
	"github.com/F1997/nightingale/pushgw/idents"
	"github.com/F1997/nightingale/storage"

	"github.com/gin-gonic/gin"
	"github.com/rakyll/statik/fs"
	"github.com/toolkits/pkg/ginx"
	"github.com/toolkits/pkg/logger"
	"github.com/toolkits/pkg/runner"
)

type Router struct {
	HTTP              httpx.Config
	Center            cconf.Center
	Operations        cconf.Operation
	DatasourceCache   *memsto.DatasourceCacheType
	NotifyConfigCache *memsto.NotifyConfigCacheType
	PromClients       *prom.PromClientMap
	Redis             storage.Redis
	MetaSet           *metas.Set
	IdentSet          *idents.Set
	TargetCache       *memsto.TargetCacheType
	Sso               *sso.SsoClient
	UserCache         *memsto.UserCacheType
	UserGroupCache    *memsto.UserGroupCacheType
	Ctx               *ctx.Context

	DatasourceCheckHook func(*gin.Context) bool
}

func New(httpConfig httpx.Config, center cconf.Center, operations cconf.Operation, ds *memsto.DatasourceCacheType, ncc *memsto.NotifyConfigCacheType,
	pc *prom.PromClientMap, redis storage.Redis, sso *sso.SsoClient, ctx *ctx.Context, metaSet *metas.Set, idents *idents.Set, tc *memsto.TargetCacheType,
	uc *memsto.UserCacheType, ugc *memsto.UserGroupCacheType) *Router {
	return &Router{
		HTTP:              httpConfig,
		Center:            center,
		Operations:        operations,
		DatasourceCache:   ds,
		NotifyConfigCache: ncc,
		PromClients:       pc,
		Redis:             redis,
		MetaSet:           metaSet,
		IdentSet:          idents,
		TargetCache:       tc,
		Sso:               sso,
		UserCache:         uc,
		UserGroupCache:    ugc,
		Ctx:               ctx,

		DatasourceCheckHook: func(ctx *gin.Context) bool { return false },
	}
}

func stat() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 请求开始处理时间
		start := time.Now()
		// 请求传递给下一个中间件或处理函数
		c.Next()
		// 获取请求的状态码、请求的 HTTP 方法和请求的路径
		code := fmt.Sprintf("%d", c.Writer.Status())
		method := c.Request.Method
		// 创建一个标签（labels）切片，其中包括以下信息：
		// cstats.Service：服务名称，可能是应用程序的名称或标识。
		// code：HTTP 响应状态码。
		// c.FullPath()：请求的完整路径。
		// method：HTTP 请求方法。
		labels := []string{cstats.Service, code, c.FullPath(), method}
		// 增加请求计数器的值。这个计数器用于统计不同类型请求的数量
		cstats.RequestCounter.WithLabelValues(labels...).Inc()
		// 记录请求的处理时间。
		cstats.RequestDuration.WithLabelValues(labels...).Observe(float64(time.Since(start).Seconds()))
	}
}

// languageDetector 中间件，用于检测和设置请求的语言标识
func languageDetector(i18NHeaderKey string) gin.HandlerFunc {
	headerKey := i18NHeaderKey
	return func(c *gin.Context) {
		if headerKey != "" {
			lang := c.GetHeader(headerKey)
			if lang != "" {
				if strings.HasPrefix(lang, "zh") {
					c.Request.Header.Set("X-Language", "zh")
				} else if strings.HasPrefix(lang, "en") {
					c.Request.Header.Set("X-Language", "en")
				} else {
					c.Request.Header.Set("X-Language", lang)
				}
			} else {
				c.Request.Header.Set("X-Language", "en")
			}
		}
		c.Next()
	}
}

func (rt *Router) configNoRoute(r *gin.Engine, fs *http.FileSystem) {
	// 如果没有匹配到任何路由时执行
	r.NoRoute(func(c *gin.Context) {
		// 解析 url，提取后缀
		arr := strings.Split(c.Request.URL.Path, ".")
		suffix := arr[len(arr)-1]

		switch suffix {
		case "png", "jpeg", "jpg", "svg", "ico", "gif", "css", "js", "html", "htm", "gz", "zip", "map", "ttf":
			if !rt.Center.UseFileAssets {
				c.FileFromFS(c.Request.URL.Path, *fs)
			} else {
				cwdarr := []string{"/"}
				if runtime.GOOS == "windows" {
					cwdarr[0] = ""
				}
				cwdarr = append(cwdarr, strings.Split(runner.Cwd, "/")...)
				cwdarr = append(cwdarr, "pub")
				cwdarr = append(cwdarr, strings.Split(c.Request.URL.Path, "/")...)
				c.File(path.Join(cwdarr...))
			}
		default:
			if !rt.Center.UseFileAssets {
				c.FileFromFS("/", *fs)
			} else {
				cwdarr := []string{"/"}
				if runtime.GOOS == "windows" {
					cwdarr[0] = ""
				}
				cwdarr = append(cwdarr, strings.Split(runner.Cwd, "/")...)
				cwdarr = append(cwdarr, "pub")
				cwdarr = append(cwdarr, "index.html")
				c.File(path.Join(cwdarr...))
			}
		}
	})
}

// 配置 Gin 框架的路由和中间件，从而定义了应用程序的不同路由和处理函数
func (rt *Router) Config(r *gin.Engine) {
	// 记录请求的统计信息
	r.Use(stat())
	// 检查请求头中指定的语言标识,语言信息存储在请求头中的 "X-Language" 字段中
	r.Use(languageDetector(rt.Center.I18NHeaderKey))
	// 恢复返回一个中间件，在出现 panic 时写入 500，防止程序崩溃
	r.Use(aop.Recovery())

	statikFS, err := fs.New()
	if err != nil {
		logger.Errorf("cannot create statik fs: %v", err)
	}

	if !rt.Center.UseFileAssets {
		// 配置静态文件服务，将请求映射到嵌入式文件系统（statikFS）上的文件
		r.StaticFS("/pub", statikFS)
	}

	// 定义路由组 /api/n9e
	pagesPrefix := "/api/n9e"
	pages := r.Group(pagesPrefix)
	{

		if rt.Center.AnonymousAccess.PromQuerier {
			pages.Any("/proxy/:id/*url", rt.dsProxy)
			pages.POST("/query-range-batch", rt.promBatchQueryRange)
			pages.POST("/query-instant-batch", rt.promBatchQueryInstant)
			pages.GET("/datasource/brief", rt.datasourceBriefs)
		} else {
			pages.Any("/proxy/:id/*url", rt.auth(), rt.dsProxy)
			pages.POST("/query-range-batch", rt.auth(), rt.promBatchQueryRange)
			pages.POST("/query-instant-batch", rt.auth(), rt.promBatchQueryInstant)
			pages.GET("/datasource/brief", rt.auth(), rt.datasourceBriefs)
		}

		pages.POST("/auth/login", rt.jwtMock(), rt.loginPost)
		pages.POST("/auth/logout", rt.jwtMock(), rt.auth(), rt.logoutPost)
		pages.POST("/auth/refresh", rt.jwtMock(), rt.refreshPost)
		pages.POST("/auth/captcha", rt.jwtMock(), rt.generateCaptcha)
		pages.POST("/auth/captcha-verify", rt.jwtMock(), rt.captchaVerify)
		pages.GET("/auth/ifshowcaptcha", rt.ifShowCaptcha)

		pages.GET("/auth/sso-config", rt.ssoConfigNameGet)
		pages.GET("/auth/rsa-config", rt.rsaConfigGet)
		pages.GET("/auth/redirect", rt.loginRedirect)
		pages.GET("/auth/redirect/cas", rt.loginRedirectCas)
		pages.GET("/auth/redirect/oauth", rt.loginRedirectOAuth)
		pages.GET("/auth/callback", rt.loginCallback)
		pages.GET("/auth/callback/cas", rt.loginCallbackCas)
		pages.GET("/auth/callback/oauth", rt.loginCallbackOAuth)
		pages.GET("/auth/perms", rt.allPerms)

		pages.GET("/metrics/desc", rt.metricsDescGetFile)
		pages.POST("/metrics/desc", rt.metricsDescGetMap)

		pages.GET("/notify-channels", rt.notifyChannelsGets)
		pages.GET("/contact-keys", rt.contactKeysGets)

		// 个人中心
		pages.GET("/self/perms", rt.auth(), rt.user(), rt.permsGets)
		pages.GET("/self/profile", rt.auth(), rt.user(), rt.selfProfileGet)
		pages.PUT("/self/profile", rt.auth(), rt.user(), rt.selfProfilePut)
		pages.PUT("/self/password", rt.auth(), rt.user(), rt.selfPasswordPut)

		// 用户列表
		pages.GET("/users", rt.auth(), rt.user(), rt.perm("/users"), rt.userGets)
		// 添加用户
		pages.POST("/users", rt.auth(), rt.admin(), rt.userAddPost)
		// 获取用户信息
		pages.GET("/user/:id/profile", rt.auth(), rt.userProfileGet)
		// 修改用户信息
		pages.PUT("/user/:id/profile", rt.auth(), rt.admin(), rt.userProfilePut)
		// 修改用户密码
		pages.PUT("/user/:id/password", rt.auth(), rt.admin(), rt.userPasswordPut)
		// 删除用户
		pages.DELETE("/user/:id", rt.auth(), rt.admin(), rt.userDel)

		pages.GET("/metric-views", rt.auth(), rt.metricViewGets)
		pages.DELETE("/metric-views", rt.auth(), rt.user(), rt.metricViewDel)
		pages.POST("/metric-views", rt.auth(), rt.user(), rt.metricViewAdd)
		pages.PUT("/metric-views", rt.auth(), rt.user(), rt.metricViewPut)

		// 团队列表
		pages.GET("/user-groups", rt.auth(), rt.user(), rt.userGroupGets)
		// 添加团队
		pages.POST("/user-groups", rt.auth(), rt.user(), rt.perm("/user-groups/add"), rt.userGroupAdd)
		// 获取团队信息
		pages.GET("/user-group/:id", rt.auth(), rt.user(), rt.userGroupGet)
		// 修改团队信息
		pages.PUT("/user-group/:id", rt.auth(), rt.user(), rt.perm("/user-groups/put"), rt.userGroupWrite(), rt.userGroupPut)
		// 删除团队
		pages.DELETE("/user-group/:id", rt.auth(), rt.user(), rt.perm("/user-groups/del"), rt.userGroupWrite(), rt.userGroupDel)
		// 添加团队成员
		pages.POST("/user-group/:id/members", rt.auth(), rt.user(), rt.perm("/user-groups/put"), rt.userGroupWrite(), rt.userGroupMemberAdd)
		// 删除团队成员
		pages.DELETE("/user-group/:id/members", rt.auth(), rt.user(), rt.perm("/user-groups/put"), rt.userGroupWrite(), rt.userGroupMemberDel)

		// 查询业务组列表
		pages.GET("/busi-groups", rt.auth(), rt.user(), rt.busiGroupGets) // 业务组列表
		// 新增业务组
		pages.POST("/busi-groups", rt.auth(), rt.user(), rt.perm("/busi-groups/add"), rt.busiGroupAdd)

		pages.GET("/busi-groups/alertings", rt.auth(), rt.busiGroupAlertingsGets)
		// 查询业务组信息
		pages.GET("/busi-group/:id", rt.auth(), rt.user(), rt.bgro(), rt.busiGroupGet)
		// 修改业务组信息
		pages.PUT("/busi-group/:id", rt.auth(), rt.user(), rt.perm("/busi-groups/put"), rt.bgrw(), rt.busiGroupPut)
		// 业务组的团队
		pages.POST("/busi-group/:id/members", rt.auth(), rt.user(), rt.perm("/busi-groups/put"), rt.bgrw(), rt.busiGroupMemberAdd)
		// 删除业务组团队
		pages.DELETE("/busi-group/:id/members", rt.auth(), rt.user(), rt.perm("/busi-groups/put"), rt.bgrw(), rt.busiGroupMemberDel)
		// 删除业务组
		pages.DELETE("/busi-group/:id", rt.auth(), rt.user(), rt.perm("/busi-groups/del"), rt.bgrw(), rt.busiGroupDel)
		pages.GET("/busi-group/:id/perm/:perm", rt.auth(), rt.user(), rt.checkBusiGroupPerm)

		// 机器列表
		pages.GET("/targets", rt.auth(), rt.user(), rt.targetGets)
		pages.POST("/target/list", rt.auth(), rt.user(), rt.targetGetsByHostFilter)
		pages.DELETE("/targets", rt.auth(), rt.user(), rt.perm("/targets/del"), rt.targetDel)
		pages.GET("/targets/tags", rt.auth(), rt.user(), rt.targetGetTags)
		pages.POST("/targets/tags", rt.auth(), rt.user(), rt.perm("/targets/put"), rt.targetBindTagsByFE)
		pages.DELETE("/targets/tags", rt.auth(), rt.user(), rt.perm("/targets/put"), rt.targetUnbindTagsByFE)
		pages.PUT("/targets/note", rt.auth(), rt.user(), rt.perm("/targets/put"), rt.targetUpdateNote)
		pages.PUT("/targets/bgid", rt.auth(), rt.user(), rt.perm("/targets/put"), rt.targetUpdateBgid)

		pages.POST("/builtin-cate-favorite", rt.auth(), rt.user(), rt.builtinCateFavoriteAdd)
		pages.DELETE("/builtin-cate-favorite/:name", rt.auth(), rt.user(), rt.builtinCateFavoriteDel)

		pages.GET("/builtin-boards", rt.builtinBoardGets)
		pages.GET("/builtin-board/:name", rt.builtinBoardGet)
		pages.GET("/dashboards/builtin/list", rt.builtinBoardGets)
		pages.GET("/builtin-boards-cates", rt.auth(), rt.user(), rt.builtinBoardCateGets)
		pages.POST("/builtin-boards-detail", rt.auth(), rt.user(), rt.builtinBoardDetailGets)
		pages.GET("/integrations/icon/:cate/:name", rt.builtinIcon)
		pages.GET("/integrations/makedown/:cate", rt.builtinMarkdown)

		pages.GET("/busi-group/:id/boards", rt.auth(), rt.user(), rt.perm("/dashboards"), rt.bgro(), rt.boardGets)
		pages.POST("/busi-group/:id/boards", rt.auth(), rt.user(), rt.perm("/dashboards/add"), rt.bgrw(), rt.boardAdd)
		pages.POST("/busi-group/:id/board/:bid/clone", rt.auth(), rt.user(), rt.perm("/dashboards/add"), rt.bgrw(), rt.boardClone)

		pages.GET("/board/:bid", rt.boardGet)
		pages.GET("/board/:bid/pure", rt.boardPureGet)
		pages.PUT("/board/:bid", rt.auth(), rt.user(), rt.perm("/dashboards/put"), rt.boardPut)
		pages.PUT("/board/:bid/configs", rt.auth(), rt.user(), rt.perm("/dashboards/put"), rt.boardPutConfigs)
		pages.PUT("/board/:bid/public", rt.auth(), rt.user(), rt.perm("/dashboards/put"), rt.boardPutPublic)
		pages.DELETE("/boards", rt.auth(), rt.user(), rt.perm("/dashboards/del"), rt.boardDel)

		pages.GET("/share-charts", rt.chartShareGets)
		pages.POST("/share-charts", rt.auth(), rt.chartShareAdd)

		// 内置规则
		pages.GET("/alert-rules/builtin/alerts-cates", rt.auth(), rt.user(), rt.builtinAlertCateGets)
		pages.GET("/alert-rules/builtin/list", rt.auth(), rt.user(), rt.builtinAlertRules)

		pages.GET("/busi-group/:id/alert-rules", rt.auth(), rt.user(), rt.perm("/alert-rules"), rt.alertRuleGets)
		pages.POST("/busi-group/:id/alert-rules", rt.auth(), rt.user(), rt.perm("/alert-rules/add"), rt.bgrw(), rt.alertRuleAddByFE)
		pages.POST("/busi-group/:id/alert-rules/import", rt.auth(), rt.user(), rt.perm("/alert-rules/add"), rt.bgrw(), rt.alertRuleAddByImport)
		pages.DELETE("/busi-group/:id/alert-rules", rt.auth(), rt.user(), rt.perm("/alert-rules/del"), rt.bgrw(), rt.alertRuleDel)
		pages.PUT("/busi-group/:id/alert-rules/fields", rt.auth(), rt.user(), rt.perm("/alert-rules/put"), rt.bgrw(), rt.alertRulePutFields)
		pages.PUT("/busi-group/:id/alert-rule/:arid", rt.auth(), rt.user(), rt.perm("/alert-rules/put"), rt.alertRulePutByFE)
		pages.GET("/alert-rule/:arid", rt.auth(), rt.user(), rt.perm("/alert-rules"), rt.alertRuleGet)
		pages.PUT("/busi-group/alert-rule/validate", rt.auth(), rt.user(), rt.perm("/alert-rules/put"), rt.alertRuleValidation)

		pages.GET("/busi-group/:id/recording-rules", rt.auth(), rt.user(), rt.perm("/recording-rules"), rt.recordingRuleGets)
		pages.POST("/busi-group/:id/recording-rules", rt.auth(), rt.user(), rt.perm("/recording-rules/add"), rt.bgrw(), rt.recordingRuleAddByFE)
		pages.DELETE("/busi-group/:id/recording-rules", rt.auth(), rt.user(), rt.perm("/recording-rules/del"), rt.bgrw(), rt.recordingRuleDel)
		pages.PUT("/busi-group/:id/recording-rule/:rrid", rt.auth(), rt.user(), rt.perm("/recording-rules/put"), rt.bgrw(), rt.recordingRulePutByFE)
		pages.GET("/recording-rule/:rrid", rt.auth(), rt.user(), rt.perm("/recording-rules"), rt.recordingRuleGet)
		pages.PUT("/busi-group/:id/recording-rules/fields", rt.auth(), rt.user(), rt.perm("/recording-rules/put"), rt.recordingRulePutFields)

		pages.GET("/busi-group/:id/alert-mutes", rt.auth(), rt.user(), rt.perm("/alert-mutes"), rt.bgro(), rt.alertMuteGetsByBG)
		pages.POST("/busi-group/:id/alert-mutes/preview", rt.auth(), rt.user(), rt.perm("/alert-mutes/add"), rt.bgrw(), rt.alertMutePreview)
		pages.POST("/busi-group/:id/alert-mutes", rt.auth(), rt.user(), rt.perm("/alert-mutes/add"), rt.bgrw(), rt.alertMuteAdd)
		pages.DELETE("/busi-group/:id/alert-mutes", rt.auth(), rt.user(), rt.perm("/alert-mutes/del"), rt.bgrw(), rt.alertMuteDel)
		pages.PUT("/busi-group/:id/alert-mute/:amid", rt.auth(), rt.user(), rt.perm("/alert-mutes/put"), rt.alertMutePutByFE)
		pages.PUT("/busi-group/:id/alert-mutes/fields", rt.auth(), rt.user(), rt.perm("/alert-mutes/put"), rt.bgrw(), rt.alertMutePutFields)

		pages.GET("/busi-group/:id/alert-subscribes", rt.auth(), rt.user(), rt.perm("/alert-subscribes"), rt.bgro(), rt.alertSubscribeGets)
		pages.GET("/alert-subscribe/:sid", rt.auth(), rt.user(), rt.perm("/alert-subscribes"), rt.alertSubscribeGet)
		pages.POST("/busi-group/:id/alert-subscribes", rt.auth(), rt.user(), rt.perm("/alert-subscribes/add"), rt.bgrw(), rt.alertSubscribeAdd)
		pages.PUT("/busi-group/:id/alert-subscribes", rt.auth(), rt.user(), rt.perm("/alert-subscribes/put"), rt.bgrw(), rt.alertSubscribePut)
		pages.DELETE("/busi-group/:id/alert-subscribes", rt.auth(), rt.user(), rt.perm("/alert-subscribes/del"), rt.bgrw(), rt.alertSubscribeDel)

		if rt.Center.AnonymousAccess.AlertDetail {
			pages.GET("/alert-cur-event/:eid", rt.alertCurEventGet)
			pages.GET("/alert-his-event/:eid", rt.alertHisEventGet)
		} else {
			pages.GET("/alert-cur-event/:eid", rt.auth(), rt.alertCurEventGet)
			pages.GET("/alert-his-event/:eid", rt.auth(), rt.alertHisEventGet)
		}

		// card logic
		pages.GET("/alert-cur-events/list", rt.auth(), rt.alertCurEventsList)
		pages.GET("/alert-cur-events/card", rt.auth(), rt.alertCurEventsCard)
		pages.POST("/alert-cur-events/card/details", rt.auth(), rt.alertCurEventsCardDetails)
		// 历史告警
		pages.GET("/alert-his-events/list", rt.auth(), rt.alertHisEventsList)

		pages.DELETE("/alert-cur-events", rt.auth(), rt.user(), rt.perm("/alert-cur-events/del"), rt.alertCurEventDel)
		pages.GET("/alert-cur-events/stats", rt.auth(), rt.alertCurEventsStatistics)

		// 聚合规则
		pages.GET("/alert-aggr-views", rt.auth(), rt.alertAggrViewGets)
		pages.DELETE("/alert-aggr-views", rt.auth(), rt.user(), rt.alertAggrViewDel)
		pages.POST("/alert-aggr-views", rt.auth(), rt.user(), rt.alertAggrViewAdd)
		pages.PUT("/alert-aggr-views", rt.auth(), rt.user(), rt.alertAggrViewPut)

		// 自愈脚本
		pages.GET("/busi-group/:id/task-tpls", rt.auth(), rt.user(), rt.perm("/job-tpls"), rt.bgro(), rt.taskTplGets)
		pages.POST("/busi-group/:id/task-tpls", rt.auth(), rt.user(), rt.perm("/job-tpls/add"), rt.bgrw(), rt.taskTplAdd)
		pages.DELETE("/busi-group/:id/task-tpl/:tid", rt.auth(), rt.user(), rt.perm("/job-tpls/del"), rt.bgrw(), rt.taskTplDel)
		pages.POST("/busi-group/:id/task-tpls/tags", rt.auth(), rt.user(), rt.perm("/job-tpls/put"), rt.bgrw(), rt.taskTplBindTags)
		pages.DELETE("/busi-group/:id/task-tpls/tags", rt.auth(), rt.user(), rt.perm("/job-tpls/put"), rt.bgrw(), rt.taskTplUnbindTags)
		pages.GET("/busi-group/:id/task-tpl/:tid", rt.auth(), rt.user(), rt.perm("/job-tpls"), rt.bgro(), rt.taskTplGet)
		pages.PUT("/busi-group/:id/task-tpl/:tid", rt.auth(), rt.user(), rt.perm("/job-tpls/put"), rt.bgrw(), rt.taskTplPut)

		// 告警自愈，执行历史
		pages.GET("/busi-group/:id/tasks", rt.auth(), rt.user(), rt.perm("/job-tasks"), rt.bgro(), rt.taskGets)
		pages.POST("/busi-group/:id/tasks", rt.auth(), rt.user(), rt.perm("/job-tasks/add"), rt.bgrw(), rt.taskAdd)
		pages.GET("/busi-group/:id/task/*url", rt.auth(), rt.user(), rt.perm("/job-tasks"), rt.taskProxy)
		pages.PUT("/busi-group/:id/task/*url", rt.auth(), rt.user(), rt.perm("/job-tasks/put"), rt.bgrw(), rt.taskProxy)

		// 告警引擎
		pages.GET("/servers", rt.auth(), rt.admin(), rt.serversGet)

		pages.GET("/server-clusters", rt.auth(), rt.admin(), rt.serverClustersGet)

		// 已接入数据源
		pages.POST("/datasource/list", rt.auth(), rt.datasourceList)
		// 默认的数据列表
		pages.POST("/datasource/plugin/list", rt.auth(), rt.pluginList)
		// 修改已接入数据源
		pages.POST("/datasource/upsert", rt.auth(), rt.admin(), rt.datasourceUpsert)
		// 已接入数据源信息
		pages.POST("/datasource/desc", rt.auth(), rt.admin(), rt.datasourceGet)
		// 禁&启用
		pages.POST("/datasource/status/update", rt.auth(), rt.admin(), rt.datasourceUpdataStatus)
		// 删除已接入数据源
		pages.DELETE("/datasource/", rt.auth(), rt.admin(), rt.datasourceDel)

		// 角色管理
		pages.GET("/roles", rt.auth(), rt.admin(), rt.roleGets)
		pages.POST("/roles", rt.auth(), rt.admin(), rt.roleAdd)
		pages.PUT("/roles", rt.auth(), rt.admin(), rt.rolePut)
		pages.DELETE("/role/:id", rt.auth(), rt.admin(), rt.roleDel)

		// 角色权限
		pages.GET("/role/:id/ops", rt.auth(), rt.admin(), rt.operationOfRole)
		pages.PUT("/role/:id/ops", rt.auth(), rt.admin(), rt.roleBindOperation)
		// 权限列表
		pages.GET("/operation", rt.operations)

		// 通知模版
		pages.GET("/notify-tpls", rt.auth(), rt.admin(), rt.notifyTplGets)
		pages.PUT("/notify-tpl/content", rt.auth(), rt.admin(), rt.notifyTplUpdateContent)
		pages.PUT("/notify-tpl", rt.auth(), rt.admin(), rt.notifyTplUpdate)
		pages.POST("/notify-tpl", rt.auth(), rt.admin(), rt.notifyTplAdd)
		pages.DELETE("/notify-tpl/:id", rt.auth(), rt.admin(), rt.notifyTplDel)
		pages.POST("/notify-tpl/preview", rt.auth(), rt.admin(), rt.notifyTplPreview)

		// 单点登录
		pages.GET("/sso-configs", rt.auth(), rt.admin(), rt.ssoConfigGets)
		pages.PUT("/sso-config", rt.auth(), rt.admin(), rt.ssoConfigUpdate)

		// 回调
		pages.GET("/webhooks", rt.auth(), rt.admin(), rt.webhookGets)
		pages.PUT("/webhooks", rt.auth(), rt.admin(), rt.webhookPuts)

		// 通知脚本
		pages.GET("/notify-script", rt.auth(), rt.admin(), rt.notifyScriptGet)
		pages.PUT("/notify-script", rt.auth(), rt.admin(), rt.notifyScriptPut)

		// 通知渠道
		pages.GET("/notify-channel", rt.auth(), rt.admin(), rt.notifyChannelGets)
		pages.PUT("/notify-channel", rt.auth(), rt.admin(), rt.notifyChannelPuts)

		// 通知列表
		pages.GET("/notify-contact", rt.auth(), rt.admin(), rt.notifyContactGets)
		pages.PUT("/notify-contact", rt.auth(), rt.admin(), rt.notifyContactPuts)

		// 自愈配置
		pages.GET("/notify-config", rt.auth(), rt.admin(), rt.notifyConfigGet)
		pages.PUT("/notify-config", rt.auth(), rt.admin(), rt.notifyConfigPut)
		// SMTP 测试
		pages.PUT("/smtp-config-test", rt.auth(), rt.admin(), rt.attemptSendEmail)

		pages.GET("/es-index-pattern", rt.auth(), rt.esIndexPatternGet)
		// 日志索引模式列表
		pages.GET("/es-index-pattern-list", rt.auth(), rt.esIndexPatternGetList)
		pages.POST("/es-index-pattern", rt.auth(), rt.admin(), rt.esIndexPatternAdd)
		pages.PUT("/es-index-pattern", rt.auth(), rt.admin(), rt.esIndexPatternPut)
		pages.DELETE("/es-index-pattern", rt.auth(), rt.admin(), rt.esIndexPatternDel)

		pages.GET("/config", rt.auth(), rt.admin(), rt.configGetByKey)
		pages.PUT("/config", rt.auth(), rt.admin(), rt.configPutByKey)
	}

	// 用于获取应用程序的版本信息
	r.GET("/api/n9e/versions", func(c *gin.Context) {
		v := version.Version
		lastIndex := strings.LastIndex(version.Version, "-")
		if lastIndex != -1 {
			v = version.Version[:lastIndex]
		}

		ginx.NewRender(c).Data(gin.H{"version": v, "github_verison": version.GithubVersion.Load().(string)}, nil)
	})

	// 用于提供面向服务的 API，包含了对 Prometheus 数据源的代理、用户管理、告警规则管理等功能
	if rt.HTTP.APIForService.Enable {
		service := r.Group("/v1/n9e")
		if len(rt.HTTP.APIForService.BasicAuth) > 0 {
			service.Use(gin.BasicAuth(rt.HTTP.APIForService.BasicAuth))
		}
		{
			service.Any("/prometheus/*url", rt.dsProxy)
			service.POST("/users", rt.userAddPost)
			service.GET("/users", rt.userFindAll)

			service.GET("/user-groups", rt.userGroupGetsByService)
			service.GET("/user-group-members", rt.userGroupMemberGetsByService)

			service.GET("/targets", rt.targetGetsByService)
			service.GET("/targets/tags", rt.targetGetTags)
			service.POST("/targets/tags", rt.targetBindTagsByService)
			service.DELETE("/targets/tags", rt.targetUnbindTagsByService)
			service.PUT("/targets/note", rt.targetUpdateNoteByService)

			service.POST("/alert-rules", rt.alertRuleAddByService)
			service.DELETE("/alert-rules", rt.alertRuleDelByService)
			service.PUT("/alert-rule/:arid", rt.alertRulePutByService)
			service.GET("/alert-rule/:arid", rt.alertRuleGet)
			service.GET("/alert-rules", rt.alertRulesGetByService)

			service.GET("/alert-subscribes", rt.alertSubscribeGetsByService)

			service.GET("/busi-groups", rt.busiGroupGetsByService)

			service.GET("/datasources", rt.datasourceGetsByService)
			service.GET("/datasource-ids", rt.getDatasourceIds)
			service.POST("/server-heartbeat", rt.serverHeartbeat)
			service.GET("/servers-active", rt.serversActive)

			service.GET("/recording-rules", rt.recordingRuleGetsByService)

			service.GET("/alert-mutes", rt.alertMuteGets)
			service.POST("/alert-mutes", rt.alertMuteAddByService)
			service.DELETE("/alert-mutes", rt.alertMuteDel)

			service.GET("/alert-cur-events", rt.alertCurEventsList)
			service.GET("/alert-cur-events-get-by-rid", rt.alertCurEventsGetByRid)
			service.GET("/alert-his-events", rt.alertHisEventsList)
			service.GET("/alert-his-event/:eid", rt.alertHisEventGet)

			service.GET("/task-tpl/:tid", rt.taskTplGetByService)

			service.GET("/config/:id", rt.configGet)
			service.GET("/configs", rt.configsGet)
			service.GET("/config", rt.configGetByKey)
			service.PUT("/configs", rt.configsPut)
			service.POST("/configs", rt.configsPost)
			service.DELETE("/configs", rt.configsDel)

			service.POST("/conf-prop/encrypt", rt.confPropEncrypt)
			service.POST("/conf-prop/decrypt", rt.confPropDecrypt)

			service.GET("/statistic", rt.statistic)

			service.GET("/notify-tpls", rt.notifyTplGets)

			service.POST("/task-record-add", rt.taskRecordAdd)
		}
	}

	if rt.HTTP.APIForAgent.Enable {
		heartbeat := r.Group("/v1/n9e")
		{
			if len(rt.HTTP.APIForAgent.BasicAuth) > 0 {
				heartbeat.Use(gin.BasicAuth(rt.HTTP.APIForAgent.BasicAuth))
			}
			heartbeat.POST("/heartbeat", rt.heartbeat)
		}
	}

	rt.configNoRoute(r, &statikFS)

}

// 根据传入的数据对象和可选的错误消息，在 Gin 框架的上下文中渲染 JSON 响应
func Render(c *gin.Context, data, msg interface{}) {
	if msg == nil {
		if data == nil {
			data = struct{}{}
		}
		c.JSON(http.StatusOK, gin.H{"data": data, "error": ""})
	} else {
		c.JSON(http.StatusOK, gin.H{"error": gin.H{"message": msg}})
	}
}

// 根据传入的错误信息或错误对象，在 Gin 框架的上下文中响应一个包含错误信息的 JSON 响应
func Dangerous(c *gin.Context, v interface{}, code ...int) {
	if v == nil {
		return
	}
	// 通过类型断言判断 v 的类型
	switch t := v.(type) {
	case string:
		if t != "" {
			c.JSON(http.StatusOK, gin.H{"error": v})
		}
	case error:
		c.JSON(http.StatusOK, gin.H{"error": t.Error()})
	}
}

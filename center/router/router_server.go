package router

import (
	"time"

	"github.com/F1997/nightingale/models"

	"github.com/gin-gonic/gin"
	"github.com/toolkits/pkg/ginx"
)

// 获取服务器列表
func (rt *Router) serversGet(c *gin.Context) {
	list, err := models.AlertingEngineGets(rt.Ctx, "")
	ginx.NewRender(c).Data(list, err)
}

// 获取集群列表
func (rt *Router) serverClustersGet(c *gin.Context) {
	list, err := models.AlertingEngineGetsClusters(rt.Ctx, "")
	ginx.NewRender(c).Data(list, err)
}

// 更新心跳信息
func (rt *Router) serverHeartbeat(c *gin.Context) {
	var req models.HeartbeatInfo
	ginx.BindJSON(c, &req)
	err := models.AlertingEngineHeartbeatWithCluster(rt.Ctx, req.Instance, req.EngineCluster, req.DatasourceId)
	ginx.NewRender(c).Message(err)
}

// 获取活跃服务器列表
func (rt *Router) serversActive(c *gin.Context) {
	datasourceId := ginx.QueryInt64(c, "dsid")
	engineName := ginx.QueryStr(c, "engine_name", "")
	if engineName != "" {
		servers, err := models.AlertingEngineGetsInstances(rt.Ctx, "engine_cluster = ? and clock > ?", engineName, time.Now().Unix()-30)
		ginx.NewRender(c).Data(servers, err)
		return
	}

	servers, err := models.AlertingEngineGetsInstances(rt.Ctx, "datasource_id = ? and clock > ?", datasourceId, time.Now().Unix()-30)
	ginx.NewRender(c).Data(servers, err)
}

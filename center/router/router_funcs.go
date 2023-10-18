package router

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/F1997/nightingale/alert/aconf"
	"github.com/F1997/nightingale/models"
	"github.com/F1997/nightingale/pkg/ctx"
	"github.com/F1997/nightingale/pkg/ibex"
	"github.com/gin-gonic/gin"

	"github.com/toolkits/pkg/ginx"
)

const defaultLimit = 300

// 用于获取不同数据对象的统计信息，根据传入的 name 参数来区分要统计的对象类型。
func (rt *Router) statistic(c *gin.Context) {
	name := ginx.QueryStr(c, "name")
	var model interface{}
	var err error
	var statistics *models.Statistics
	switch name {
	case "alert_mute":
		model = models.AlertMute{}
	case "alert_rule":
		model = models.AlertRule{}
	case "alert_subscribe":
		model = models.AlertSubscribe{}
	case "busi_group":
		model = models.BusiGroup{}
	case "recording_rule":
		model = models.RecordingRule{}
	case "target":
		model = models.Target{}
	case "user":
		model = models.User{}
	case "user_group":
		model = models.UserGroup{}
	case "datasource":
		// datasource update_at is different from others
		statistics, err = models.DatasourceStatistics(rt.Ctx)
		ginx.NewRender(c).Data(statistics, err)
		return
	default:
		ginx.Bomb(http.StatusBadRequest, "invalid name")
	}

	statistics, err = models.StatisticsGet(rt.Ctx, model)
	ginx.NewRender(c).Data(statistics, err)
}

// 从请求中提取并解析数据源的ID
func queryDatasourceIds(c *gin.Context) []int64 {
	datasourceIds := ginx.QueryStr(c, "datasource_ids", "")
	datasourceIds = strings.ReplaceAll(datasourceIds, ",", " ")
	idsStr := strings.Fields(datasourceIds)
	ids := make([]int64, len(idsStr))
	for i, idStr := range idsStr {
		id, _ := strconv.ParseInt(idStr, 10, 64)
		ids[i] = id
	}
	return ids
}

// 用于验证包含ID的表单数据
type idsForm struct {
	Ids []int64 `json:"ids"`
}

// 包含ID的表单数据验证方法
func (f idsForm) Verify() {
	if len(f.Ids) == 0 {
		ginx.Bomb(http.StatusBadRequest, "ids empty")
	}
}

// 根据用户ID获取用户对象
func User(ctx *ctx.Context, id int64) *models.User {
	obj, err := models.UserGetById(ctx, id)
	ginx.Dangerous(err)

	if obj == nil {
		ginx.Bomb(http.StatusNotFound, "No such user")
	}

	return obj
}

// 根据团队ID获取团队对象。
func UserGroup(ctx *ctx.Context, id int64) *models.UserGroup {
	obj, err := models.UserGroupGetById(ctx, id)
	ginx.Dangerous(err)

	if obj == nil {
		ginx.Bomb(http.StatusNotFound, "No such UserGroup")
	}

	return obj
}

// 根据业务组ID获取业务组对象。
func BusiGroup(ctx *ctx.Context, id int64) *models.BusiGroup {
	obj, err := models.BusiGroupGetById(ctx, id)
	ginx.Dangerous(err)

	if obj == nil {
		ginx.Bomb(http.StatusNotFound, "No such BusiGroup")
	}

	return obj
}

// 根据仪表板ID获取仪表板对象。
func Dashboard(ctx *ctx.Context, id int64) *models.Dashboard {
	obj, err := models.DashboardGet(ctx, "id=?", id)
	ginx.Dangerous(err)

	if obj == nil {
		ginx.Bomb(http.StatusNotFound, "No such dashboard")
	}

	return obj
}

type DoneIdsReply struct {
	Err string `json:"err"`
	Dat struct {
		List []int64 `json:"list"`
	} `json:"dat"`
}

type TaskCreateReply struct {
	Err string `json:"err"`
	Dat int64  `json:"dat"` // task.id
}

// 用于创建任务， return task.id, error
func TaskCreate(v interface{}, ibexc aconf.Ibex) (int64, error) {
	var res TaskCreateReply
	err := ibex.New(
		ibexc.Address,
		ibexc.BasicAuthUser,
		ibexc.BasicAuthPass,
		ibexc.Timeout,
	).
		Path("/ibex/v1/tasks").
		In(v).
		Out(&res).
		POST()

	if err != nil {
		return 0, err
	}

	if res.Err != "" {
		return 0, fmt.Errorf("response.err: %v", res.Err)
	}

	return res.Dat, nil
}

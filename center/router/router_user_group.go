package router

import (
	"net/http"
	"time"

	"github.com/F1997/nightingale/models"

	"github.com/gin-gonic/gin"
	"github.com/toolkits/pkg/ginx"
	"github.com/toolkits/pkg/logger"
)

func (rt *Router) checkBusiGroupPerm(c *gin.Context) {
	// 当前登陆用户信息
	me := c.MustGet("user").(*models.User)
	// 需要检查权限的业务组ID
	bg := BusiGroup(rt.Ctx, ginx.UrlParamInt64(c, "id"))

	// 检查当前用户是否具有执行该操作的权限
	can, err := me.CanDoBusiGroup(rt.Ctx, bg, ginx.UrlParamStr(c, "perm"))
	// 返回数据
	ginx.NewRender(c).Data(can, err)
}

func (rt *Router) userGroupGets(c *gin.Context) {
	limit := ginx.QueryInt(c, "limit", 1500)
	query := ginx.QueryStr(c, "query", "")

	me := c.MustGet("user").(*models.User)
	lst, err := me.UserGroups(rt.Ctx, limit, query)

	ginx.NewRender(c).Data(lst, err)
}

func (rt *Router) userGroupGetsByService(c *gin.Context) {
	lst, err := models.UserGroupGetAll(rt.Ctx)
	ginx.NewRender(c).Data(lst, err)
}

// user group member get by service
func (rt *Router) userGroupMemberGetsByService(c *gin.Context) {
	members, err := models.UserGroupMemberGetAll(rt.Ctx)
	ginx.NewRender(c).Data(members, err)
}

type userGroupForm struct {
	Name string `json:"name" binding:"required"`
	Note string `json:"note"`
}

// 用户组添加
func (rt *Router) userGroupAdd(c *gin.Context) {
	var f userGroupForm
	ginx.BindJSON(c, &f)

	me := c.MustGet("user").(*models.User)

	ug := models.UserGroup{
		Name:     f.Name,
		Note:     f.Note,
		CreateBy: me.Username,
		UpdateBy: me.Username,
	}

	err := ug.Add(rt.Ctx)
	if err == nil {
		// Even failure is not a big deal
		models.UserGroupMemberAdd(rt.Ctx, ug.Id, me.Id)
	}

	ginx.NewRender(c).Data(ug.Id, err)
}

func (rt *Router) userGroupPut(c *gin.Context) {
	var f userGroupForm
	ginx.BindJSON(c, &f)

	me := c.MustGet("user").(*models.User)
	ug := c.MustGet("user_group").(*models.UserGroup)

	if ug.Name != f.Name {
		// name changed, check duplication
		num, err := models.UserGroupCount(rt.Ctx, "name=? and id<>?", f.Name, ug.Id)
		ginx.Dangerous(err)

		if num > 0 {
			ginx.Bomb(http.StatusOK, "UserGroup already exists")
		}
	}

	ug.Name = f.Name
	ug.Note = f.Note
	ug.UpdateBy = me.Username
	ug.UpdateAt = time.Now().Unix()

	ginx.NewRender(c).Message(ug.Update(rt.Ctx, "Name", "Note", "UpdateAt", "UpdateBy"))
}

// Return all members, front-end search and paging
func (rt *Router) userGroupGet(c *gin.Context) {
	ug := UserGroup(rt.Ctx, ginx.UrlParamInt64(c, "id"))

	ids, err := models.MemberIds(rt.Ctx, ug.Id)
	ginx.Dangerous(err)

	logger.Info("userGroupGet", ids)
	users, err := models.UserGetsByIds(rt.Ctx, ids)

	ginx.NewRender(c).Data(gin.H{
		"users":      users,
		"user_group": ug,
	}, err)
}

func (rt *Router) userGroupDel(c *gin.Context) {
	ug := c.MustGet("user_group").(*models.UserGroup)
	ginx.NewRender(c).Message(ug.Del(rt.Ctx))
}

func (rt *Router) userGroupMemberAdd(c *gin.Context) {
	var f idsForm
	ginx.BindJSON(c, &f)
	f.Verify()

	me := c.MustGet("user").(*models.User)
	ug := c.MustGet("user_group").(*models.UserGroup)

	err := ug.AddMembers(rt.Ctx, f.Ids)
	if err == nil {
		ug.UpdateAt = time.Now().Unix()
		ug.UpdateBy = me.Username
		ug.Update(rt.Ctx, "UpdateAt", "UpdateBy")
	}

	ginx.NewRender(c).Message(err)
}

func (rt *Router) userGroupMemberDel(c *gin.Context) {
	var f idsForm
	ginx.BindJSON(c, &f)
	f.Verify()

	me := c.MustGet("user").(*models.User)
	ug := c.MustGet("user_group").(*models.UserGroup)

	err := ug.DelMembers(rt.Ctx, f.Ids)
	if err == nil {
		ug.UpdateAt = time.Now().Unix()
		ug.UpdateBy = me.Username
		ug.Update(rt.Ctx, "UpdateAt", "UpdateBy")
	}

	ginx.NewRender(c).Message(err)
}

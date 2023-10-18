package router

import (
	"github.com/F1997/nightingale/models"
	"github.com/F1997/nightingale/pkg/ormx"

	"github.com/gin-gonic/gin"
	"github.com/toolkits/pkg/ginx"
)

// 获取当前用户的个人信息
func (rt *Router) selfProfileGet(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	if user.IsAdmin() {
		user.Admin = true
	}
	ginx.NewRender(c).Data(user, nil)
}

// 定义了用户更新个人资料时所需的字段，包括昵称、电话、电子邮件、头像和联系方式
type selfProfileForm struct {
	Nickname string       `json:"nickname"`
	Phone    string       `json:"phone"`
	Email    string       `json:"email"`
	Portrait string       `json:"portrait"`
	Contacts ormx.JSONObj `json:"contacts"`
}

// 更新当前用户的个人资料
func (rt *Router) selfProfilePut(c *gin.Context) {
	var f selfProfileForm
	ginx.BindJSON(c, &f)

	user := c.MustGet("user").(*models.User)
	user.Nickname = f.Nickname
	user.Phone = f.Phone
	user.Email = f.Email
	user.Portrait = f.Portrait
	user.Contacts = f.Contacts
	user.UpdateBy = user.Username

	ginx.NewRender(c).Message(user.UpdateAllFields(rt.Ctx))
}

// 定义了用户修改密码时所需的字段，包括旧密码和新密码。
type selfPasswordForm struct {
	OldPass string `json:"oldpass" binding:"required"`
	NewPass string `json:"newpass" binding:"required"`
}

// 修改密码
func (rt *Router) selfPasswordPut(c *gin.Context) {
	var f selfPasswordForm
	ginx.BindJSON(c, &f)
	user := c.MustGet("user").(*models.User)
	ginx.NewRender(c).Message(user.ChangePassword(rt.Ctx, f.OldPass, f.NewPass))
}

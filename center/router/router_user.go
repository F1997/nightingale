package router

import (
	"net/http"
	"strings"

	"github.com/F1997/nightingale/models"
	"github.com/F1997/nightingale/pkg/ormx"

	"github.com/gin-gonic/gin"
	"github.com/toolkits/pkg/ginx"
)

func (rt *Router) userFindAll(c *gin.Context) {
	// 获取所有用户
	list, err := models.UserGetAll(rt.Ctx)
	// 渲染并返回用户列表数据
	ginx.NewRender(c).Data(list, err)
}

func (rt *Router) userGets(c *gin.Context) {
	limit := ginx.QueryInt(c, "limit", 20)
	query := ginx.QueryStr(c, "query", "")

	// 查询符合条件的总数
	total, err := models.UserTotal(rt.Ctx, query)
	// 处理错误
	ginx.Dangerous(err)

	// 查询符合条件的用户列表
	list, err := models.UserGets(rt.Ctx, query, limit, ginx.Offset(c, limit))
	ginx.Dangerous(err)

	// 获取当前登录用户信息
	user := c.MustGet("user").(*models.User)

	// 创建渲染器，将数据渲染为 JSON 格式，为响应发送给客户端
	ginx.NewRender(c).Data(gin.H{
		"list":  list,
		"total": total,
		"admin": user.IsAdmin(),
	}, nil)
}

// 添加用户表单
type userAddForm struct {
	Username string       `json:"username" binding:"required"`
	Password string       `json:"password" binding:"required"`
	Nickname string       `json:"nickname"`
	Phone    string       `json:"phone"`
	Email    string       `json:"email"`
	Portrait string       `json:"portrait"`
	Roles    []string     `json:"roles" binding:"required"`
	Contacts ormx.JSONObj `json:"contacts"`
}

// 处理用户添加请求
func (rt *Router) userAddPost(c *gin.Context) {
	var f userAddForm
	// 对json 数据绑定到 f
	ginx.BindJSON(c, &f)

	// 对密码进行加密
	password, err := models.CryptoPass(rt.Ctx, f.Password)
	ginx.Dangerous(err)

	// 角色信息是否为空
	if len(f.Roles) == 0 {
		ginx.Bomb(http.StatusBadRequest, "roles empty")
	}

	// 获取当前登录用户信息
	user := c.MustGet("user").(*models.User)

	u := models.User{
		Username: f.Username,
		Password: password,
		Nickname: f.Nickname,
		Phone:    f.Phone,
		Email:    f.Email,
		Portrait: f.Portrait,
		Roles:    strings.Join(f.Roles, " "),
		Contacts: f.Contacts,
		CreateBy: user.Username,
		UpdateBy: user.Username,
	}

	// u.Add(rt.Ctx) 添加新用户，ginx.NewRender(c).Message() 返回结果
	ginx.NewRender(c).Message(u.Add(rt.Ctx))
}

func (rt *Router) userProfileGet(c *gin.Context) {
	// 获取指定ID 用户的个人资料
	user := User(rt.Ctx, ginx.UrlParamInt64(c, "id"))
	// json 返回客户端
	ginx.NewRender(c).Data(user, nil)
}

type userProfileForm struct {
	Nickname string       `json:"nickname"`
	Phone    string       `json:"phone"`
	Email    string       `json:"email"`
	Roles    []string     `json:"roles"`
	Contacts ormx.JSONObj `json:"contacts"`
}

func (rt *Router) userProfilePut(c *gin.Context) {
	var f userProfileForm
	ginx.BindJSON(c, &f)

	if len(f.Roles) == 0 {
		ginx.Bomb(http.StatusBadRequest, "roles empty")
	}

	target := User(rt.Ctx, ginx.UrlParamInt64(c, "id"))
	target.Nickname = f.Nickname
	target.Phone = f.Phone
	target.Email = f.Email
	target.Roles = strings.Join(f.Roles, " ")
	target.Contacts = f.Contacts
	target.UpdateBy = c.MustGet("username").(string)

	ginx.NewRender(c).Message(target.UpdateAllFields(rt.Ctx))
}

type userPasswordForm struct {
	Password string `json:"password" binding:"required"`
}

func (rt *Router) userPasswordPut(c *gin.Context) {
	var f userPasswordForm
	ginx.BindJSON(c, &f)

	target := User(rt.Ctx, ginx.UrlParamInt64(c, "id"))

	cryptoPass, err := models.CryptoPass(rt.Ctx, f.Password)
	ginx.Dangerous(err)

	ginx.NewRender(c).Message(target.UpdatePassword(rt.Ctx, cryptoPass, c.MustGet("username").(string)))
}

func (rt *Router) userDel(c *gin.Context) {
	id := ginx.UrlParamInt64(c, "id")
	target, err := models.UserGetById(rt.Ctx, id)
	ginx.Dangerous(err)

	if target == nil {
		ginx.NewRender(c).Message(nil)
		return
	}

	ginx.NewRender(c).Message(target.Del(rt.Ctx))
}

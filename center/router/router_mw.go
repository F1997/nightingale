package router

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/F1997/nightingale/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/toolkits/pkg/ginx"
)

type AccessDetails struct {
	AccessUuid   string
	UserIdentity string
}

// 该函数 handleProxyUser 是在一个 Router 对象上定义的。该函数负责处理与代理用户身份验证相关的请求。
func (rt *Router) handleProxyUser(c *gin.Context) *models.User {
	// 获取 headerUserNameKey
	headerUserNameKey := rt.HTTP.ProxyAuth.HeaderUserNameKey
	// 使用 Gin 上下文的 GetHeader 方法根据 headerUserNameKey 从 HTTP 请求头中获取用户名
	username := c.GetHeader(headerUserNameKey)
	if username == "" {
		ginx.Bomb(http.StatusUnauthorized, "unauthorized")
	}

	// 通过模型查询用户是否存在
	user, err := models.UserGetByUsername(rt.Ctx, username)
	if err != nil {
		ginx.Bomb(http.StatusInternalServerError, err.Error())
	}

	// 如果 user 不存在，创建用户
	if user == nil {
		now := time.Now().Unix()
		user = &models.User{
			Username: username,
			Nickname: username,
			Roles:    strings.Join(rt.HTTP.ProxyAuth.DefaultRoles, " "),
			CreateAt: now,
			UpdateAt: now,
			CreateBy: "system",
			UpdateBy: "system",
		}
		err = user.Add(rt.Ctx)
		if err != nil {
			ginx.Bomb(http.StatusInternalServerError, err.Error())
		}
	}
	return user
}

// 调用 rt.handleProxyUser(c) 方法来处理代理用户的身份验证。rt.handleProxyUser(c) 返回用户模型对象，表示代理用户
func (rt *Router) proxyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := rt.handleProxyUser(c)
		// 将用户的 ID 和用户名设置到 Gin 上下文
		c.Set("userid", user.Id)
		c.Set("username", user.Username)
		// 传递给下一个中间件或请求处理函数，允许请求继续处理。
		c.Next()
	}
}

// 处理 JSON Web Token (JWT) 身份验证
func (rt *Router) jwtAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 提取 JWT 的元数据信息
		metadata, err := rt.extractTokenMetadata(c.Request)
		if err != nil {
			// 抛出 HTTP 401 未授权的错误并中止请求处理
			ginx.Bomb(http.StatusUnauthorized, "unauthorized")
		}

		// 获取用户身份信息
		userIdentity, err := rt.fetchAuth(c.Request.Context(), metadata.AccessUuid)
		if err != nil {
			// 抛出 HTTP 401 未授权的错误并中止请求处理
			ginx.Bomb(http.StatusUnauthorized, "unauthorized")
		}

		// 通过字符串分割操作将用户身份信息拆分成用户ID和用户名，格式 ${userid}-${username} 的格式保存在 JWT 中
		// ${userid}-${username}
		arr := strings.SplitN(userIdentity, "-", 2)
		if len(arr) != 2 {
			ginx.Bomb(http.StatusUnauthorized, "unauthorized")
		}

		// 将用户ID (userid) 解析为整数类型
		userid, err := strconv.ParseInt(arr[0], 10, 64)
		if err != nil {
			ginx.Bomb(http.StatusUnauthorized, "unauthorized")
		}

		// 将用户ID和用户名存储在 Gin 上下文中，以便后续的请求处理函数可以访问这些信息
		c.Set("userid", userid)
		c.Set("username", arr[1])

		// 调用 c.Next() 将控制传递给下一个中间件或请求处理函数，允许请求继续处理
		c.Next()
	}
}

func (rt *Router) auth() gin.HandlerFunc {
	// 检查 rt.HTTP.ProxyAuth.Enable 的值，这个值表示是否启用了代理身份验证。
	// 如果启用了代理身份验证，则返回 rt.proxyAuth() 的结果，即使用代理身份验证的中间件
	if rt.HTTP.ProxyAuth.Enable {
		return rt.proxyAuth()
	} else {
		return rt.jwtAuth()
	}
}

// 模拟 JWT 登录、注销和刷新请求，但仅在代理身份验证启用时生效
// if proxy auth is enabled, mock jwt login/logout/refresh request
func (rt *Router) jwtMock() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查 rt.HTTP.ProxyAuth.Enable 的值，以确定代理身份验证是否启用
		if !rt.HTTP.ProxyAuth.Enable {
			// 请求传递给下一个中间件或请求处理函数，结束中间件的执行
			c.Next()
			return
		}
		// 判断请求的路径是否包含 "logout"
		if strings.Contains(c.FullPath(), "logout") {
			// 抛出一个 HTTP 400 错误响应，响应内容 "logout" 不受支持
			ginx.Bomb(http.StatusBadRequest, "logout is not supported when proxy auth is enabled")
		}
		// 调用 rt.handleProxyUser(c) 方法来处理代理用户的身份验证，并获取代理用户的信息
		user := rt.handleProxyUser(c)
		// 构建一个响应数据，其中包括用户信息、访问令牌和刷新令牌
		// 访问令牌和刷新令牌都为空字符串，因为它只是模拟请求，不实际生成令牌
		ginx.NewRender(c).Data(gin.H{
			"user":          user,
			"access_token":  "",
			"refresh_token": "",
		}, nil)
		// 中止请求的继续处理，确保后续中间件或请求处理函数不会再次处理该请求
		c.Abort()
	}
}

func (rt *Router) user() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取存储在 Gin 上下文中的用户ID，断言为 int64 类型
		userid := c.MustGet("userid").(int64)
		// 从数据库中根据用户ID获取用户信息
		user, err := models.UserGetById(rt.Ctx, userid)
		if err != nil {
			// 抛出 HTTP 401 未授权的错误并中止请求处理
			ginx.Bomb(http.StatusUnauthorized, "unauthorized")
		}

		if user == nil {
			ginx.Bomb(http.StatusUnauthorized, "unauthorized")
		}

		// 用户信息和管理员状态存储在 Gin 上下文
		c.Set("user", user)
		c.Set("isadmin", user.IsAdmin())
		// 传递给下一个中间件或请求处理函数
		c.Next()
	}
}

// 用于检查当前用户是否有权限修改指定的用户组
func (rt *Router) userGroupWrite() gin.HandlerFunc {
	return func(c *gin.Context) {
		//  获取存储在 Gin 上下文中的用户信息，并将其断言为 *models.User 类型
		me := c.MustGet("user").(*models.User)
		// 调用 ginx.UrlParamInt64(c, "id") 从请求的 URL 参数中获取用户组的ID
		// 使用 UserGroup(rt.Ctx, ginx.UrlParamInt64(c, "id")) 方法，它获取指定用户组的信息，并将其存储在变量 ug 中
		ug := UserGroup(rt.Ctx, ginx.UrlParamInt64(c, "id"))

		// 检查当前用户是否有权限修改指定的用户组
		can, err := me.CanModifyUserGroup(rt.Ctx, ug)
		// 错误处理函数
		ginx.Dangerous(err)

		// 用户没有权限修改用户组，它会使用 ginx.Bomb 函数回复一个 HTTP 403 禁止访问的错误，并中止请求处理
		if !can {
			ginx.Bomb(http.StatusForbidden, "forbidden")
		}
		// 用户组信息存储在 Gin 上下文中，以供后续的请求处理函数使用，存储在键 "user_group" 下
		c.Set("user_group", ug)
		// 传递给下一个中间件或请求处理函数
		c.Next()
	}
}

// 检查当前用户是否有权限执行特定业务组的操作
func (rt *Router) bgro() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取存储在 Gin 上下文中的用户信息，并将其断言为 *models.User 类型
		me := c.MustGet("user").(*models.User)
		// 调用 ginx.UrlParamInt64(c, "id") 从请求的 URL 参数中获取业务组的ID
		// 获取信息，并将其存储在变量 bg 中
		bg := BusiGroup(rt.Ctx, ginx.UrlParamInt64(c, "id"))

		// 检查当前用户是否有权限执行特定业务组的操作,
		can, err := me.CanDoBusiGroup(rt.Ctx, bg)
		ginx.Dangerous(err)

		// 没有权限执行操作，返回一个 HTTP 403 禁止访问的错误，并中止请求处理
		if !can {
			ginx.Bomb(http.StatusForbidden, "forbidden")
		}
		// 将业务组信息存储在 Gin 上下文中，以供后续的请求处理函数使用，存储在键 "busi_group" 下
		c.Set("busi_group", bg)
		// 传递给下一个中间件或请求处理函数
		c.Next()
	}
}

// bgrw 逐步要被干掉，不安全
func (rt *Router) bgrw() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取存储在 Gin 上下文中的用户信息，并将其断言为 *models.User 类型
		me := c.MustGet("user").(*models.User)
		// 从请求的 URL 参数中获取业务组的ID
		// 获取指定业务组的信息，并将其存储在变量 bg 中
		bg := BusiGroup(rt.Ctx, ginx.UrlParamInt64(c, "id"))
		// 检查当前用户是否有权限执行
		can, err := me.CanDoBusiGroup(rt.Ctx, bg, "rw")
		// 如果出现错误，它会使用 ginx.Dangerous 函数处理错误
		ginx.Dangerous(err)
		// 如果用户没有权限，返回一个 HTTP 403 禁止访问的错误
		if !can {
			ginx.Bomb(http.StatusForbidden, "forbidden")
		}
		// 将 bg 存储在 Gin 上下文中的 busi_group 键下
		c.Set("busi_group", bg)
		// 传递给下一个中间件或请求处理函数
		c.Next()
	}
}

// bgrwCheck 要逐渐替换掉bgrw方法，更安全
func (rt *Router) bgrwCheck(c *gin.Context, bgid int64) {
	// 获取用户信息，断言为 *models.User 类型
	me := c.MustGet("user").(*models.User)
	// 通过 bgid 取指定业务组的信息，并将其存储在变量 bg 中
	bg := BusiGroup(rt.Ctx, bgid)
	// 检查当前用户是否有权限
	can, err := me.CanDoBusiGroup(rt.Ctx, bg, "rw")
	ginx.Dangerous(err)
	// 如果用户没有权限，返回一个 HTTP 403 禁止访问的错误
	if !can {
		ginx.Bomb(http.StatusForbidden, "forbidden")
	}
	// 将 bg 存储在 Gin 上下文中的 busi_group 键下
	c.Set("busi_group", bg)
}

func (rt *Router) bgrwChecks(c *gin.Context, bgids []int64) {
	// 创建 map 用于存放已经检查过的 bgid
	set := make(map[int64]struct{})
	// 循环遍历 bgids
	for i := 0; i < len(bgids); i++ {
		// 判断 bgid 是否已经在 map中
		if _, has := set[bgids[i]]; has {
			continue
		}
		// 调用 bgrwCheck()
		rt.bgrwCheck(c, bgids[i])
		// 已经检查过的存入 map
		set[bgids[i]] = struct{}{}
	}
}

func (rt *Router) bgroCheck(c *gin.Context, bgid int64) {
	me := c.MustGet("user").(*models.User)
	bg := BusiGroup(rt.Ctx, bgid)

	can, err := me.CanDoBusiGroup(rt.Ctx, bg)
	ginx.Dangerous(err)

	if !can {
		ginx.Bomb(http.StatusForbidden, "forbidden")
	}
	c.Set("busi_group", bg)
}

func (rt *Router) perm(operation string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户信息，断言为 *models.User 类型
		me := c.MustGet("user").(*models.User)

		// 检查当前用户是否具有执行特定操作权限
		can, err := me.CheckPerm(rt.Ctx, operation)
		ginx.Dangerous(err)
		// 没有权限，返回 HTTP 403 禁止访问的错误，并中止请求处理
		if !can {
			ginx.Bomb(http.StatusForbidden, "forbidden")
		}
		// 传递给下一个中间件或请求处理函数
		c.Next()
	}
}

func (rt *Router) admin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取 userid，断言为int64
		userid := c.MustGet("userid").(int64)

		// 通过 userid 获取用户信息
		user, err := models.UserGetById(rt.Ctx, userid)
		if err != nil {
			// 返回 HTTP 401
			ginx.Bomb(http.StatusUnauthorized, "unauthorized")
		}

		if user == nil {
			// 返回 HTTP 401
			ginx.Bomb(http.StatusUnauthorized, "unauthorized")
		}
		// 将用户的角色信息从 user.Roles 字符串中分割为一个角色切片 roles
		roles := strings.Fields(user.Roles)
		found := false
		// 遍历 roles
		for i := 0; i < len(roles); i++ {
			// 如果有对应的 AdminRole，设置found为true
			if roles[i] == models.AdminRole {
				found = true
				break
			}
		}

		if !found {
			// 返回 HTTP 403
			ginx.Bomb(http.StatusForbidden, "forbidden")
		}
		// 将用户信息存储在 Gin 上下文中的 ‘user’ 下
		c.Set("user", user)
		// 控制传递给下一个中间件或请求处理函数
		c.Next()
	}
}

func (rt *Router) extractTokenMetadata(r *http.Request) (*AccessDetails, error) {
	// rt.extractToken(r) 从请求中提取 JWT 令牌
	// rt.verifyToken 验证 JWT 令牌的有效性，使用的是 rt.HTTP.JWTAuth.SigningKey 作为密钥进行验证。
	// 验证通过，存储在 token 变量中
	token, err := rt.verifyToken(rt.HTTP.JWTAuth.SigningKey, rt.extractToken(r))
	if err != nil {
		return nil, err
	}
	// token.Claims 解析为 jwt.MapClaims 类型，并检查是否有效
	claims, ok := token.Claims.(jwt.MapClaims)
	if ok && token.Valid {
		// 提取 access_uuid 和 user_identity 字段的值
		accessUuid, ok := claims["access_uuid"].(string)
		if !ok {
			return nil, errors.New("failed to parse access_uuid from jwt")
		}
		// 返回 AccessDetails 结构体指针
		return &AccessDetails{
			AccessUuid:   accessUuid,
			UserIdentity: claims["user_identity"].(string),
		}, nil
	}

	return nil, err
}

func (rt *Router) extractToken(r *http.Request) string {
	// 从请求头部获取名为 "Authorization" 的头部字段的值
	tok := r.Header.Get("Authorization")
	// 检查 tok 的长度是否大于 6，并且头部字段的值是否以 "BEARER " 开头（不区分大小写）
	if len(tok) > 6 && strings.ToUpper(tok[0:7]) == "BEARER " {
		// 返回从第 7 个字符开始的子字符串，即去除了 "BEARER " 前缀的部分，作为提取到的 JWT 令牌
		return tok[7:]
	}

	return ""
}

func (rt *Router) createAuth(ctx context.Context, userIdentity string, td *TokenDetails) error {
	// 转换为时间对象，表示访问令牌和刷新令牌的过期时间
	at := time.Unix(td.AtExpires, 0)
	rte := time.Unix(td.RtExpires, 0)
	now := time.Now()
	// 通过 Set 方法将用户身份信息和令牌关联起来存储在 Redis 中。
	// rt.wrapJwtKey(td.AccessUuid) 返回一个包装过的键，用于存储访问令牌。
	// userIdentity 是与令牌相关联的用户身份信息。
	// 过期时间是根据令牌的过期时间计算的相对差值，以确保在指定的时间后令牌自动过期。
	errAccess := rt.Redis.Set(ctx, rt.wrapJwtKey(td.AccessUuid), userIdentity, at.Sub(now)).Err()
	if errAccess != nil {
		return errAccess
	}
	// rt.wrapJwtKey(td.RefreshUuid) 返回一个包装过的键，用于存储刷新令牌。
	errRefresh := rt.Redis.Set(ctx, rt.wrapJwtKey(td.RefreshUuid), userIdentity, rte.Sub(now)).Err()
	if errRefresh != nil {
		return errRefresh
	}

	return nil
}

func (rt *Router) fetchAuth(ctx context.Context, givenUuid string) (string, error) {
	// 通过 Get 方法从 Redis 中检索与给定 UUID 相关联的用户身份信息
	return rt.Redis.Get(ctx, rt.wrapJwtKey(givenUuid)).Result()
}

func (rt *Router) deleteAuth(ctx context.Context, givenUuid string) error {
	// 通过 Del 方法从 Redis 中删除与给定 UUID 相关联的键值对
	return rt.Redis.Del(ctx, rt.wrapJwtKey(givenUuid)).Err()
}

// 删除访问令牌和刷新令牌
func (rt *Router) deleteTokens(ctx context.Context, authD *AccessDetails) error {
	// get the refresh uuid
	refreshUuid := authD.AccessUuid + "++" + authD.UserIdentity

	// delete access token
	err := rt.Redis.Del(ctx, rt.wrapJwtKey(authD.AccessUuid)).Err()
	if err != nil {
		return err
	}

	// delete refresh token
	err = rt.Redis.Del(ctx, rt.wrapJwtKey(refreshUuid)).Err()
	if err != nil {
		return err
	}

	return nil
}

// 包装 jwtkey
func (rt *Router) wrapJwtKey(key string) string {
	return rt.HTTP.JWTAuth.RedisKeyPrefix + key
}

// token 结构体
type TokenDetails struct {
	AccessToken  string // 访问令牌
	RefreshToken string // 刷新令牌
	AccessUuid   string // 访问令牌的唯一标识符
	RefreshUuid  string // 刷新令牌的唯一标识符
	AtExpires    int64  // 访问令牌的过期时间戳
	RtExpires    int64  // 刷新令牌的过期时间戳
}

func (rt *Router) createTokens(signingKey, userIdentity string) (*TokenDetails, error) {
	// 创建空的 TokenDetails 结构体实例 td
	td := &TokenDetails{}
	// 计算访问令牌的过期时间
	td.AtExpires = time.Now().Add(time.Minute * time.Duration(rt.HTTP.JWTAuth.AccessExpired)).Unix()
	// 创建访问令牌的唯一标识符
	td.AccessUuid = uuid.NewString()
	// 计算刷新令牌的过期时间
	td.RtExpires = time.Now().Add(time.Minute * time.Duration(rt.HTTP.JWTAuth.RefreshExpired)).Unix()
	// 创建刷新令牌的唯一标识符
	td.RefreshUuid = td.AccessUuid + "++" + userIdentity

	var err error
	// Creating Access Token
	atClaims := jwt.MapClaims{}
	atClaims["authorized"] = true
	atClaims["access_uuid"] = td.AccessUuid
	atClaims["user_identity"] = userIdentity
	atClaims["exp"] = td.AtExpires
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)
	td.AccessToken, err = at.SignedString([]byte(signingKey))
	if err != nil {
		return nil, err
	}

	// Creating Refresh Token
	rtClaims := jwt.MapClaims{}
	rtClaims["refresh_uuid"] = td.RefreshUuid
	rtClaims["user_identity"] = userIdentity
	rtClaims["exp"] = td.RtExpires
	jrt := jwt.NewWithClaims(jwt.SigningMethodHS256, rtClaims)
	td.RefreshToken, err = jrt.SignedString([]byte(signingKey))
	if err != nil {
		return nil, err
	}

	return td, nil
}

// 用于验证 JWT 令牌的有效性并解析其内容
func (rt *Router) verifyToken(signingKey, tokenString string) (*jwt.Token, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("bearer token not found")
	}
	// 使用 jwt.Parse 函数来解析 JWT 令牌， 回调函数：
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// 检查 JWT 的签名方法是否是 *jwt.SigningMethodHMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected jwt signing method: %v", token.Header["alg"])
		}
		// 返回签名密钥
		return []byte(signingKey), nil
	})
	if err != nil {
		return nil, err
	}
	// 返回 *jwt.Token
	return token, nil
}

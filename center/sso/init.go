package sso

import (
	"log"

	"github.com/BurntSushi/toml"
	"github.com/F1997/nightingale/center/cconf"
	"github.com/F1997/nightingale/models"
	"github.com/F1997/nightingale/pkg/cas"
	"github.com/F1997/nightingale/pkg/ctx"
	"github.com/F1997/nightingale/pkg/ldapx"
	"github.com/F1997/nightingale/pkg/oauth2x"
	"github.com/F1997/nightingale/pkg/oidcx"

	"github.com/toolkits/pkg/logger"
)

// 定义了一个 SsoClient 结构体，用于存储不同类型的 SSO 客户端配置，包括 OIDC、LDAP、CAS 和 OAuth2
type SsoClient struct {
	OIDC   *oidcx.SsoClient
	LDAP   *ldapx.SsoClient
	CAS    *cas.SsoClient
	OAuth2 *oauth2x.SsoClient
}

// LDAP 配置信息
const LDAP = `
Enable = false
Host = 'ldap.example.org'
Port = 389
BaseDn = 'dc=example,dc=org'
BindUser = 'cn=manager,dc=example,dc=org'
BindPass = '*******'
# openldap format e.g. (&(uid=%s))
# AD format e.g. (&(sAMAccountName=%s))
AuthFilter = '(&(uid=%s))'
CoverAttributes = true
TLS = false
StartTLS = true
DefaultRoles = ['Standard']

[Attributes]
Nickname = 'cn'
Phone = 'mobile'
Email = 'mail'
`

// OAuth2 配置信息
const OAuth2 = `
Enable = false
DisplayName = 'OAuth2登录'
RedirectURL = 'http://127.0.0.1:18000/callback/oauth'
SsoAddr = 'https://sso.example.com/oauth2/authorize'
TokenAddr = 'https://sso.example.com/oauth2/token'
UserInfoAddr = 'https://api.example.com/api/v1/user/info'
TranTokenMethod = 'header'
ClientId = ''
ClientSecret = ''
CoverAttributes = true
DefaultRoles = ['Standard']
UserinfoIsArray = false
UserinfoPrefix = 'data'
Scopes = ['profile', 'email', 'phone']

[Attributes]
Username = 'username'
Nickname = 'nickname'
Phone = 'phone_number'
Email = 'email'
`

// CAS 配置信息
const CAS = `
Enable = false
SsoAddr = 'https://cas.example.com/cas/'
RedirectURL = 'http://127.0.0.1:18000/callback/cas'
DisplayName = 'CAS登录'
CoverAttributes = false
DefaultRoles = ['Standard']

[Attributes]
Nickname = 'nickname'
Phone = 'phone_number'
Email = 'email'
`

// OIDC 配置信息
const OIDC = `
Enable = false
DisplayName = 'OIDC登录'
RedirectURL = 'http://n9e.com/callback'
SsoAddr = 'http://sso.example.org'
ClientId = ''
ClientSecret = ''
CoverAttributes = true
DefaultRoles = ['Standard']

[Attributes]
Nickname = 'nickname'
Phone = 'phone_number'
Email = 'email'
`

// 初始化和配置不同类型的单点登录客户端
func Init(center cconf.Center, ctx *ctx.Context) *SsoClient {
	// 创建一个SsoClient
	ssoClient := new(SsoClient)
	// 创建一个字符串映射 m，其中包含不同类型 SSO 客户端的配置信息
	m := make(map[string]string)
	m["LDAP"] = LDAP
	m["CAS"] = CAS
	m["OIDC"] = OIDC
	m["OAuth2"] = OAuth2

	// 遍历 m 中的每个配置项，检查是否已经存在与数据库中，如果不存在，则将其写入数据库中。
	for name, config := range m {
		count, err := models.SsoConfigCountByName(ctx, name)
		if err != nil {
			logger.Error(err)
			continue
		}

		if count > 0 {
			continue
		}

		ssoConfig := models.SsoConfig{
			Name:    name,
			Content: config,
		}

		err = ssoConfig.Create(ctx)
		if err != nil {
			log.Fatalln(err)
		}
	}
	// 从数据库中获取所有 SSO 配置项
	configs, err := models.SsoConfigGets(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	// 根据配置项的名称进行分别处理
	for _, cfg := range configs {
		switch cfg.Name {
		case "LDAP": // 解析 TOML 配置并初始化 LDAP 客户端
			var config ldapx.Config
			err := toml.Unmarshal([]byte(cfg.Content), &config)
			if err != nil {
				log.Fatalln("init ldap failed", err)
			}
			ssoClient.LDAP = ldapx.New(config)
		case "OIDC": // 解析 TOML 配置并初始化 OIDC 客户端。
			var config oidcx.Config
			err := toml.Unmarshal([]byte(cfg.Content), &config)
			if err != nil {
				log.Fatalln("init oidc failed:", err)
			}
			oidcClient, err := oidcx.New(config)
			if err != nil {
				logger.Error("init oidc failed:", err)
			} else {
				ssoClient.OIDC = oidcClient
			}
		case "CAS": // 解析 TOML 配置并初始化 CAS 客户端
			var config cas.Config
			err := toml.Unmarshal([]byte(cfg.Content), &config)
			if err != nil {
				log.Fatalln("init cas failed:", err)
			}
			ssoClient.CAS = cas.New(config)
		case "OAuth2": // 解析 TOML 配置并初始化 OAuth2 客户端
			var config oauth2x.Config
			err := toml.Unmarshal([]byte(cfg.Content), &config)
			if err != nil {
				log.Fatalln("init oauth2 failed:", err)
			}
			ssoClient.OAuth2 = oauth2x.New(config)
		}
	}
	// 返回一个 SsoClient 结构，其中包含了初始化后的各种 SSO 客户端对象
	return ssoClient
}

package router

import (
	"github.com/F1997/nightingale/pkg/secu"

	"github.com/gin-gonic/gin"
	"github.com/toolkits/pkg/ginx"
)

type confPropCrypto struct {
	Data string `json:"data" binding:"required"`
	Key  string `json:"key" binding:"required"`
}

func (rt *Router) confPropEncrypt(c *gin.Context) {
	// 创建 confPropCrypto 结构体 f
	var f confPropCrypto
	// 将 json 数据绑定到 f
	ginx.BindJSON(c, &f)

	// 获取密钥的长度，并进行长度检查，确保它是16、24或32字节
	k := len(f.Key)
	switch k {
	default:
		c.String(400, "The key length should be 16, 24 or 32")
		return
	case 16, 24, 32:
		break
	}
	// 使用密钥 f.Key 对敏感数据 f.Data 进行加密
	s, err := secu.DealWithEncrypt(f.Data, f.Key)
	if err != nil {
		c.String(500, err.Error())
	}

	// 以 JSON 格式返回原始数据、密钥和加密后的数据。
	c.JSON(200, gin.H{
		"src":     f.Data,
		"key":     f.Key,
		"encrypt": s,
	})
}

func (rt *Router) confPropDecrypt(c *gin.Context) {
	var f confPropCrypto
	ginx.BindJSON(c, &f)

	k := len(f.Key)
	switch k {
	default:
		c.String(400, "The key length should be 16, 24 or 32")
		return
	case 16, 24, 32:
		break
	}

	s, err := secu.DealWithDecrypt(f.Data, f.Key)
	if err != nil {
		c.String(500, err.Error())
	}

	c.JSON(200, gin.H{
		"src":     f.Data,
		"key":     f.Key,
		"decrypt": s,
	})
}

package router

import (
	"encoding/json"

	"github.com/F1997/nightingale/models"

	"github.com/gin-gonic/gin"
	"github.com/toolkits/pkg/ginx"
)

// 获取通知渠道的标签和密钥信息
func (rt *Router) notifyChannelsGets(c *gin.Context) {
	// 创建 labelAndKeys 切片，存储通知渠道的标签和密钥信息
	var labelAndKeys []models.LabelAndKey
	// 获取与通知渠道相关的信息
	cval, err := models.ConfigsGet(rt.Ctx, models.NOTIFYCHANNEL)
	ginx.Dangerous(err)

	if cval == "" {
		// 返回一个空的 JSON 响应
		ginx.NewRender(c).Data(labelAndKeys, nil)
		return
	}

	// 创建 notifyChannels 切片，存储通知渠道的配置信息
	var notifyChannels []models.NotifyChannel
	// 通知渠道的配置信息反序列化为结构体切片
	err = json.Unmarshal([]byte(cval), &notifyChannels)
	ginx.Dangerous(err)

	// 遍历解析后的通知渠道配置
	for _, v := range notifyChannels {
		// 渠道隐藏，则跳过
		if v.Hide {
			continue
		}
		var labelAndKey models.LabelAndKey
		labelAndKey.Label = v.Name
		labelAndKey.Key = v.Ident
		labelAndKeys = append(labelAndKeys, labelAndKey)
	}
	// 将 labelAndKeys 切片作为数据返回给客户端，nil 代表没有错误
	ginx.NewRender(c).Data(labelAndKeys, nil)
}

// 获取联系人密钥的标签和密钥信息
func (rt *Router) contactKeysGets(c *gin.Context) {
	var labelAndKeys []models.LabelAndKey
	cval, err := models.ConfigsGet(rt.Ctx, models.NOTIFYCONTACT)
	ginx.Dangerous(err)

	if cval == "" {
		ginx.NewRender(c).Data(labelAndKeys, nil)
		return
	}

	var notifyContacts []models.NotifyContact
	err = json.Unmarshal([]byte(cval), &notifyContacts)
	ginx.Dangerous(err)

	for _, v := range notifyContacts {
		if v.Hide {
			continue
		}
		var labelAndKey models.LabelAndKey
		labelAndKey.Label = v.Name
		labelAndKey.Key = v.Ident
		labelAndKeys = append(labelAndKeys, labelAndKey)
	}

	ginx.NewRender(c).Data(labelAndKeys, nil)
}

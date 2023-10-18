package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"

	"github.com/F1997/nightingale/center/cconf"
	"github.com/F1997/nightingale/models"
	"github.com/F1997/nightingale/pkg/tplx"

	"github.com/gin-gonic/gin"
	"github.com/toolkits/pkg/ginx"
	"github.com/toolkits/pkg/str"
)

// 获取通知模板列表
func (rt *Router) notifyTplGets(c *gin.Context) {
	m := make(map[string]struct{})
	for _, channel := range models.DefaultChannels {
		m[channel] = struct{}{}
	}
	m[models.EmailSubject] = struct{}{}

	// 查询数据库中的通知模板列表，并检查每个模板是否为内置模板。
	lst, err := models.NotifyTplGets(rt.Ctx)
	for i := 0; i < len(lst); i++ {
		if _, exists := m[lst[i].Channel]; exists {
			lst[i].BuiltIn = true
		}
	}

	ginx.NewRender(c).Data(lst, err)
}

// 更新通知模板的内容
func (rt *Router) notifyTplUpdateContent(c *gin.Context) {
	var f models.NotifyTpl
	ginx.BindJSON(c, &f)
	ginx.Dangerous(templateValidate(f))

	// 更新通知模板的内容
	ginx.NewRender(c).Message(f.UpdateContent(rt.Ctx))
}

// 更新通知模板
func (rt *Router) notifyTplUpdate(c *gin.Context) {
	var f models.NotifyTpl
	ginx.BindJSON(c, &f)
	ginx.Dangerous(templateValidate(f))

	ginx.NewRender(c).Message(f.Update(rt.Ctx))
}

// 验证通知模板的有效性
func templateValidate(f models.NotifyTpl) error {
	// 检查 f.Channel 长度是否超过了 32 个字符，如果超过了，则返回错误
	if len(f.Channel) > 32 {
		return fmt.Errorf("channel length should not exceed 32")
	}
	// 检查 f.Channel 中是否包含危险字符，如果包含，则返回错误
	if str.Dangerous(f.Channel) {
		return fmt.Errorf("channel should not contain dangerous characters")
	}

	// 检查 f.Name 是否符合要求
	if len(f.Name) > 255 {
		return fmt.Errorf("name length should not exceed 255")
	}
	// 检查 f.Name 是否包含危险字符，如果包含，则返回错误
	if str.Dangerous(f.Name) {
		return fmt.Errorf("name should not contain dangerous characters")
	}

	if f.Content == "" {
		return nil
	}

	// 定义了一个字符串切片 defs，其中包含了一些默认的模板定义
	var defs = []string{
		"{{$labels := .TagsMap}}",     // 声明一个名为 $labels 的变量，该变量可以在模板中使用，并赋值为 .TagsMap
		"{{$value := .TriggerValue}}", // 声明一个名为 $value 的变量，该变量可以在模板中使用，并赋值为 .TriggerValue
	}
	// 将默认的模板定义与传入的通知模板内容连接在一起，形成一个完整的模板文本。
	text := strings.Join(append(defs, f.Content), "")

	// 创建了一个新的 Go 模板，并使用 tplx.TemplateFuncMap 中定义的自定义函数
	if _, err := template.New(f.Channel).Funcs(tplx.TemplateFuncMap).Parse(text); err != nil {
		return fmt.Errorf("notify template verify illegal:%s", err.Error())
	}

	return nil
}

// 预览通知模板的渲染效果
func (rt *Router) notifyTplPreview(c *gin.Context) {

	// 创建 models.AlertCurEvent 实例
	var event models.AlertCurEvent
	// 解析数据并填充到 event
	err := json.Unmarshal([]byte(cconf.EVENT_EXAMPLE), &event)
	ginx.Dangerous(err)

	// // 创建 models.NotifyTpl 实例，并从请求的 JSON 数据中绑定到该实例
	var f models.NotifyTpl
	ginx.BindJSON(c, &f)

	// 定义了一个字符串切片 defs，其中包含了一些默认的模板定义
	var defs = []string{
		"{{$labels := .TagsMap}}",
		"{{$value := .TriggerValue}}",
	}
	// 将默认的模板定义与传入的通知模板内容连接在一起，形成一个完整的模板文本。
	text := strings.Join(append(defs, f.Content), "")
	// 创建了一个新的 Go 模板，并使用 tplx.TemplateFuncMap 中定义的自定义函数，解析完整的模板文本
	tpl, err := template.New(f.Channel).Funcs(tplx.TemplateFuncMap).Parse(text)
	ginx.Dangerous(err)

	event.TagsMap = make(map[string]string)
	// 遍历事件数据中的标签
	for i := 0; i < len(event.TagsJSON); i++ {
		pair := strings.TrimSpace(event.TagsJSON[i])
		if pair == "" {
			continue
		}

		arr := strings.Split(pair, "=")
		if len(arr) != 2 {
			continue
		}

		event.TagsMap[arr[0]] = arr[1]
	}

	// body 用于存储模板渲染后的内容，ret 用于存储最终的渲染结果
	var body bytes.Buffer
	var ret string
	// 使用之前创建的模板 tpl，将事件数据 event 渲染到 body 变量中。
	if err := tpl.Execute(&body, event); err != nil {
		ret = err.Error()
	} else {
		// 将渲染后的内容存储在 ret 变量
		ret = body.String()
	}
	// 将渲染结果或错误信息作为响应数据返回给客户端
	ginx.NewRender(c).Data(ret, nil)
}

// add new notify template
func (rt *Router) notifyTplAdd(c *gin.Context) {
	var f models.NotifyTpl
	ginx.BindJSON(c, &f)
	f.Channel = strings.TrimSpace(f.Channel)
	ginx.Dangerous(templateValidate(f))

	count, err := models.NotifyTplCountByChannel(rt.Ctx, f.Channel)
	ginx.Dangerous(err)
	if count != 0 {
		ginx.Bomb(200, "Refuse to create duplicate channel(unique)")
	}
	ginx.NewRender(c).Message(f.Create(rt.Ctx))
}

// delete notify template, not allowed to delete the system defaults(models.DefaultChannels)
func (rt *Router) notifyTplDel(c *gin.Context) {
	f := new(models.NotifyTpl)
	id := ginx.UrlParamInt64(c, "id")
	ginx.NewRender(c).Message(f.NotifyTplDelete(rt.Ctx, id))
}

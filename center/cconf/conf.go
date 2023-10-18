package cconf

type Center struct {
	Plugins                []Plugin
	MetricsYamlFile        string          // Metrics 配置文件 路径
	OpsYamlFile            string          // 操作信息配置文件的路径
	BuiltinIntegrationsDir string          // 内置集成的目录路径
	I18NHeaderKey          string          // 国际化（i18n）的请求头键值
	MetricDesc             MetricDescType  // 指标的描述信息
	AnonymousAccess        AnonymousAccess // 匿名访问的权限配置
	UseFileAssets          bool            // 是否使用文件资源
}

type Plugin struct {
	Id       int64  `json:"id"`
	Category string `json:"category"`
	Type     string `json:"plugin_type"`
	TypeName string `json:"plugin_type_name"`
}

type AnonymousAccess struct {
	PromQuerier bool // Prometheus查询器
	AlertDetail bool // 告警详情
}

// 用于在配置加载之前进行一些预检查
func (c *Center) PreCheck() {
	if len(c.Plugins) == 0 {
		// 设置为全局变量 Plugins 的值
		c.Plugins = Plugins
	}
}

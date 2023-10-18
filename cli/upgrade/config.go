package upgrade

import (
	"bytes"
	"path"

	"github.com/F1997/nightingale/pkg/cfg"
	"github.com/F1997/nightingale/pkg/ormx"
	"github.com/F1997/nightingale/pkg/tlsx"
	"github.com/koding/multiconfig"
)

// 用于存储整体配置信息
type Config struct {
	DB       ormx.DBConfig
	Clusters []ClusterOptions
}

// 用于存储单个集群的配置信息
type ClusterOptions struct {
	Name string
	Prom string

	BasicAuthUser string
	BasicAuthPass string

	Headers []string

	Timeout     int64
	DialTimeout int64

	UseTLS bool
	tlsx.ClientConfig

	MaxIdleConnsPerHost int
}

// 解析配置文件，并将解析结果存储在 configPtr 参数指定的结构体中
func Parse(fpath string, configPtr interface{}) error {
	var (
		tBuf []byte
	)
	// 配置加载器的列表
	loaders := []multiconfig.Loader{
		&multiconfig.TagLoader{},         // 标签加载器
		&multiconfig.EnvironmentLoader{}, // 环境变量加载器
	}
	// 配置文件扫描器, 用于读取配置文件的内容
	s := cfg.NewFileScanner()

	// 读取指定路径的配置文件内容
	s.Read(path.Join(fpath))
	// 存储配置文件内容的缓冲区
	tBuf = append(tBuf, s.Data()...)
	tBuf = append(tBuf, []byte("\n")...)

	if s.Err() != nil {
		return s.Err()
	}

	if len(tBuf) != 0 {
		loaders = append(loaders, &multiconfig.TOMLLoader{Reader: bytes.NewReader(tBuf)})
	}

	// 创建一个多重加载器
	m := multiconfig.DefaultLoader{
		Loader:    multiconfig.MultiLoader(loaders...),
		Validator: multiconfig.MultiValidator(&multiconfig.RequiredValidator{}), // 属性验证器
	}
	// 将配置文件内容解析为指定结构体,并执行必需属性验证
	return m.Load(configPtr)
}

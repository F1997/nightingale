package cconf

import (
	"path"

	"github.com/toolkits/pkg/file"
)

var Operations = Operation{}

// 包含多个 Ops 结构体的切片
type Operation struct {
	Ops []Ops `yaml:"ops"`
}

type Ops struct {
	Name  string   `yaml:"name" json:"name"`
	Cname string   `yaml:"cname" json:"cname"`
	Ops   []string `yaml:"ops" json:"ops"`
}

// 从指定的 YAML 文件中加载操作信息，并将其存储到全局变量 Operations 中
func LoadOpsYaml(configDir string, opsYamlFile string) error {
	fp := opsYamlFile
	if fp == "" {
		fp = path.Join(configDir, "ops.yaml")
	}
	if !file.IsExist(fp) {
		return nil
	}
	return file.ReadYaml(fp, &Operations)
}

// 获取所有操作的列表
func GetAllOps(ops []Ops) []string {
	var ret []string
	for _, op := range ops {
		ret = append(ret, op.Ops...)
	}
	return ret
}

package conf

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/F1997/nightingale/alert/aconf"
	"github.com/F1997/nightingale/center/cconf"
	"github.com/F1997/nightingale/pkg/cfg"
	"github.com/F1997/nightingale/pkg/httpx"
	"github.com/F1997/nightingale/pkg/logx"
	"github.com/F1997/nightingale/pkg/ormx"
	"github.com/F1997/nightingale/pushgw/pconf"
	"github.com/F1997/nightingale/storage"
)

// 存储 Nightingale 的各种配置信息
type ConfigType struct {
	Global    GlobalConfig        // 全局配置
	Log       logx.Config         // 日志配置
	HTTP      httpx.Config        // HTTP 配置
	DB        ormx.DBConfig       // 数据库配置
	Redis     storage.RedisConfig // Redis 配置
	CenterApi CenterApi           // CenterApi 配置

	Pushgw pconf.Pushgw // 推送网关配置
	Alert  aconf.Alert  // 告警配置
	Center cconf.Center // 中心配置
}

// CenterApi 相关的配置
type CenterApi struct {
	Addrs         []string // CenterApi 的地址
	BasicAuthUser string   // BasicAuth 用户名
	BasicAuthPass string   // BasicAuth 密码
	Timeout       int64    // 超时时间
}

// 全局配置
type GlobalConfig struct {
	RunMode string // 运行模式
}

// 初始化 Nightingale 的配置
func InitConfig(configDir, cryptoKey string) (*ConfigType, error) {
	var config = new(ConfigType)

	// 从配置文件目录中加载配置文件
	if err := cfg.LoadConfigByDir(configDir, config); err != nil {
		return nil, fmt.Errorf("failed to load configs of directory: %s error: %s", configDir, err)
	}

	// 对推送网关、告警和中心的配置进行预检查
	config.Pushgw.PreCheck()
	config.Alert.PreCheck(configDir)
	config.Center.PreCheck()

	// 对配置文件进行解密，使用 cryptoKey 密钥进行解密
	err := decryptConfig(config, cryptoKey)
	if err != nil {
		return nil, err
	}

	if config.Alert.Heartbeat.IP == "" {
		// auto detect
		config.Alert.Heartbeat.IP = fmt.Sprint(GetOutboundIP())
		if config.Alert.Heartbeat.IP == "" {
			hostname, err := os.Hostname()
			if err != nil {
				fmt.Println("failed to get hostname:", err)
				os.Exit(1)
			}

			if strings.Contains(hostname, "localhost") {
				fmt.Println("Warning! hostname contains substring localhost, setting a more unique hostname is recommended")
			}

			config.Alert.Heartbeat.IP = hostname
		}
	}

	// 创建告警心跳端点
	config.Alert.Heartbeat.Endpoint = fmt.Sprintf("%s:%d", config.Alert.Heartbeat.IP, config.HTTP.Port)

	return config, nil
}

func GetOutboundIP() net.IP {
	// 创建udp连接，连接阿里公共dns
	conn, err := net.Dial("udp", "223.5.5.5:80")
	if err != nil {
		fmt.Println("auto get outbound ip fail:", err)
		return []byte{}
	}
	defer conn.Close()
	// 通过获取连接的本地地址信息
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	// 返回获取到的本地 IP 地址
	return localAddr.IP
}

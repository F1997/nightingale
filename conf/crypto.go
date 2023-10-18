package conf

import (
	"fmt"

	"github.com/F1997/nightingale/pkg/secu"
)

// 解密 Nightingale 配置中的敏感信息
func decryptConfig(config *ConfigType, cryptoKey string) error {
	// 密配置中的数据库连接字符串
	decryptDsn, err := secu.DealWithDecrypt(config.DB.DSN, cryptoKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt the db dsn: %s", err)
	}

	config.DB.DSN = decryptDsn

	// 循环遍历配置中的 HTTP Basic Auth 密码（APIForService）
	for k := range config.HTTP.APIForService.BasicAuth {
		decryptPwd, err := secu.DealWithDecrypt(config.HTTP.APIForService.BasicAuth[k], cryptoKey)
		if err != nil {
			return fmt.Errorf("failed to decrypt http basic auth password: %s", err)
		}

		config.HTTP.APIForService.BasicAuth[k] = decryptPwd
	}
	// 循环遍历配置中的 HTTP Basic Auth 密码（APIForAgent）
	for k := range config.HTTP.APIForAgent.BasicAuth {
		decryptPwd, err := secu.DealWithDecrypt(config.HTTP.APIForAgent.BasicAuth[k], cryptoKey)
		if err != nil {
			return fmt.Errorf("failed to decrypt http basic auth password: %s", err)
		}

		config.HTTP.APIForAgent.BasicAuth[k] = decryptPwd
	}
	// 循环遍历配置中的推送网关（PushGW）的写入者（Writers）列表
	for i, v := range config.Pushgw.Writers {
		decryptWriterPwd, err := secu.DealWithDecrypt(v.BasicAuthPass, cryptoKey)
		if err != nil {
			return fmt.Errorf("failed to decrypt writer basic auth password: %s", err)
		}

		config.Pushgw.Writers[i].BasicAuthPass = decryptWriterPwd
	}

	return nil
}

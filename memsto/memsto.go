package memsto

import (
	"os"

	"github.com/toolkits/pkg/logger"
)

// TODO 优化 exit 处理方式
func exit(code int) {
	logger.Close() // 关闭日志记录
	os.Exit(code)  // 终止当前程序的执行并返回给定的状态码
}

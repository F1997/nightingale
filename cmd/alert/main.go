package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/F1997/nightingale/alert"
	"github.com/F1997/nightingale/pkg/osx"
	"github.com/F1997/nightingale/pkg/version"

	"github.com/toolkits/pkg/runner"
)

// 用 flag 包来解析命令行参数
var (
	showVersion = flag.Bool("version", false, "Show version.")
	configDir   = flag.String("configs", osx.GetEnv("N9E_ALERT_CONFIGS", "etc"), "Specify configuration directory.(env:N9E_ALERT_CONFIGS)")
	cryptoKey   = flag.String("crypto-key", "", "Specify the secret key for configuration file field encryption.")
)

func main() {
	// 解析命令行参数
	flag.Parse()

	if *showVersion {
		fmt.Println(version.Version)
		os.Exit(0)
	}

	// 输出一些环境信息
	printEnv()

	// 初始化应用程序，传递了配置文件目录和加密密钥作为参数
	cleanFunc, err := alert.Initialize(*configDir, *cryptoKey)
	if err != nil {
		log.Fatalln("failed to initialize:", err)
	}

	code := 1
	// 创建一个信号通道 sc，用于捕获系统信号
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

EXIT:
	for {
		sig := <-sc
		fmt.Println("received signal:", sig.String())
		switch sig {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			code = 0
			break EXIT
		case syscall.SIGHUP:
			// reload configuration?
		default:
			break EXIT
		}
	}
	// 执行清理操作
	cleanFunc()
	fmt.Println("process exited")
	os.Exit(code)
}

// 输出一些环境信息
func printEnv() {
	runner.Init()
	fmt.Println("runner.cwd:", runner.Cwd)              // 当前工作目录
	fmt.Println("runner.hostname:", runner.Hostname)    // 主机名
	fmt.Println("runner.fd_limits:", runner.FdLimits()) // 文件描述符
	fmt.Println("runner.vm_limits:", runner.VMLimits()) // 虚拟内存限制
}

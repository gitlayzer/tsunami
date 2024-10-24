package signals

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/gitlayzer/tsunami/pkg/config"
)

var ShutDownSignals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}

// SetupSignalHandler 设置信号处理机制, 但不设计具体的处理操作
// 具体的处理操作需要 fn 参数传入
func SetupSignalHandler(fn func(*config.CmdOpts, chan<- bool), cmdOpts *config.CmdOpts, doneCh chan<- bool) {
	// sigCh 接收信号, 注意之后的清理操作有可能失败, 失败后不能直接退出.
	sigCh := make(chan os.Signal, 1)

	// 一般 delete pod 时, 收到的是 SIGTERM 信号.
	signal.Notify(sigCh, ShutDownSignals...)

	go func() {
		// 等待信号
		<-sigCh

		// 调用 fn(), 执行真正的清理工作
		fn(cmdOpts, doneCh)
		os.Exit(1)
	}()
}

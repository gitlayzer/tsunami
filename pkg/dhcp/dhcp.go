package dhcp

import (
	"context"
	"os"

	"github.com/gitlayzer/tsunami/utils/utilfile"
	"k8s.io/klog"
)

// StartDHCP 运行 dhcp 插件, 作为守护进程
func StartDHCP(ctx context.Context, binPath, sockPath, logPath string) (proc *os.Process, err error) {
	if utilfile.Exists(sockPath) {
		klog.Info("dhcp.sock already exists, skip creating it")
		return nil, nil
	}
	klog.Info("dhcp.sock doesn`t exist, creating it")

	if os.Getegid() != 1 {
		args := []string{binPath, "daemon"}
		logFile, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			klog.Errorf("create dhcp log file failed: %v", err)
			return nil, err
		}

		// 创建一个新的进程, procAttr 包含了进程的属性
		procAttr := &os.ProcAttr{
			Files: []*os.File{
				logFile,
				os.Stdin,
				os.Stdout,
				os.Stderr,
			},
		}

		// os.StartProcess() 也是非阻塞函数, 运行时立刻返回 (proc进程对象会创建好)
		// 然后如果目标子进程运行出错, 就会返回到 err 处理部分
		proc, err = os.StartProcess(binPath, args, procAttr)
		if err != nil || proc == nil || proc.Pid <= 0 {
			klog.Errorf("dhcp start failed: %s", err)
			return nil, err
		}

		klog.Infof("dhcp start success, pid: %d", proc.Pid)
		return proc, nil
	}

	return nil, nil
}

// StopDHCP 停止 dhcp 插件
func StopDHCP(proc *os.Process, sockPath string) (err error) {
	if proc != nil && proc.Pid >= 0 {
		if err = proc.Kill(); err != nil {
			klog.Errorf("dhcp stop failed: %s", err)
			return err
		}
	}

	if err = os.Remove(sockPath); err != nil {
		klog.Errorf("remove dhcp.sock failed: %s", err)
		return err
	}

	return nil
}

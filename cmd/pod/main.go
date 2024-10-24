package main

import (
	"context"
	"flag"
	"os"

	"github.com/gitlayzer/tsunami/pkg/bridge"
	"github.com/gitlayzer/tsunami/pkg/config"
	"github.com/gitlayzer/tsunami/pkg/dhcp"
	"github.com/gitlayzer/tsunami/pkg/signals"
	"k8s.io/klog"
)

var (
	cmdOpts        config.CmdOpts
	netConf        config.NetConf
	cmdFlags       = flag.NewFlagSet("cni-tsunami", flag.ExitOnError)
	dhcpBinPath    = "/opt/cni/bin/dhcp"
	dhcpSockPath   = "/run/cni/dhcp.sock"
	dhcpLogPath    = "/run/cni/dhcp.log"
	dhcpProc       *os.Process
	cniNetConfPath = "/etc/cni/net.d/10-cni-tsunami.conf"
)

func init() {
	cmdFlags.StringVar(&cmdOpts.Eth0Name, "iface", "", "the network interface using to communicate with kubernetes cluster")
	cmdFlags.StringVar(&cmdOpts.BridgeName, "bridge", "mybr0", "this plugin will create a bridge device, named by this option")
	cmdFlags.Parse(os.Args[1:])
}

// stopHandler 执行退出时的清理操作, 如停止dhcp进程, 恢复原本的网络拓扑等.
func stopHandler(cmdOpts *config.CmdOpts, doneCh chan<- bool) {
	var err error
	klog.Infof("receive stop signal")

	err = dhcp.StopDHCP(dhcpProc, dhcpSockPath)
	if err != nil {
		klog.Errorf("receive signal, but stop dhcp process failed: %s", err)
	}

	err = bridge.UninstallBridgeNetwork(cmdOpts.BridgeName, cmdOpts.Eth0Name)
	if err != nil {
		klog.Errorf("receive signal, but uninstall bridge network failed, you should check it: %s", err)
	}
	doneCh <- true
}

func main() {
	var err error
	klog.Info("Starting tsunami pod plugin")
	err = cmdOpts.Complete()
	if err != nil {
		klog.Error(err)
		return
	}
	klog.Infof("cmd opt: %+v", cmdOpts)

	err = netConf.Complete(cniNetConfPath)
	if err != nil {
		klog.Error(err)
		return
	}


	err = bridge.InstallBridgeNetwork(cmdOpts.BridgeName, cmdOpts.Eth0Name)
	if err != nil {
		return
	}
	klog.Info("link bridge success")

	dhcpProc, err = dhcp.StartDHCP(context.Background(), dhcpBinPath, dhcpSockPath, dhcpLogPath)
	if err != nil {
		klog.Errorf("faliled to run dhcp plugin: %s", err)
		return
	}
	klog.Info("run dhcp plugin success")

	// 退出的时机由doneCh决定.
	doneCh := make(chan bool, 1)
	signals.SetupSignalHandler(stopHandler, &cmdOpts, doneCh)
	<-doneCh

	klog.Info("exiting")
}

package config

import (
	"k8s.io/klog"

	"github.com/vishvananda/netlink"

	"github.com/gitlayzer/tsunami/pkg/cninet"
)

// CmdOpts 命令行参数对象
// 这个结构体不需要构建函数, 由 main 入口程序通过 flag 标准库自动填充
type CmdOpts struct {
	// 网桥名称
	BridgeName string
	// 集群之间通信所使用的主网卡
	// 如果不是多网卡环境, 一般是 eth0 或者 ens33
	Eth0Name string
}

// Complete 使用默认值补全 CmdOpts 对象中未指定的选项
func (c *CmdOpts) Complete() (err error) {
	// 如果未显式指定目标网络接口, 则尝试通过宿主机的默认路由获取其绑定的接口
	if c.Eth0Name == "" {
		klog.Info("doesn`t specify main network interface, try to find it")

		// 获取默认路由
		r, err := cninet.GetDefaultRoute()
		if err != nil {
			return err
		}

		// 根据路由获取绑定的网络接口
		link, err := netlink.LinkByIndex(r.LinkIndex)
		if err != nil {
			return err
		}

		// 设置网卡名称
		c.Eth0Name = link.Attrs().Name
	}

	return
}

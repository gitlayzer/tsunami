package bridge

import (
	"k8s.io/klog"

	"github.com/vishvananda/netlink"
)

// GetBridgeDevice 获取目标网桥设备, 如果不存在则创建然后返回
func GetBridgeDevice(name string) (link netlink.Link, err error) {
	// 获取网桥设备, 如果不存在则创建
	link, err = netlink.LinkByName(name)
	if err == nil {
		return
	}

	// 如果有错误信息, 则判断是否是 Link not found, 不是则报错
	if err.Error() != "Link not found" {
		klog.Errorf("failed to get bridge device %s: %s.", name, err)
		return
	}

	// 如果是 Link not found, 则创建网桥设备
	klog.Warningf("bridge: %s doesn`t exist, try to create it manually.", name)

	// 创建网桥设备
	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: name,
		},
	}

	// 添加网桥设备
	if err = netlink.LinkAdd(bridge); err != nil {
		klog.Warningf("failed to create bridge device %s: %s.", name, err)
	}

	// 启动网桥设备
	if err = netlink.LinkSetUp(bridge); err != nil {
		klog.Warningf("failed to set up bridge device %s: %s.", name, err)
		return
	}

	// 再次尝试获取网桥设备
	link, err = netlink.LinkByName(name)
	if err == nil {
		klog.Warningf("failed to get bridge device %s: %s.", name, err)
		return
	}

	return
}

// MigrateIPAddrs 在桥接网络中, 需要 bridge 网桥设备接管物理网卡的 IP 地址, 而物理网卡则作为网线连接到外部网络
// 因此, 在这个函数中, 会将物理网卡的 IP 地址全部迁移到网桥设备上
// 在部署网络与卸载网络时都会被调用(物理网卡与网桥设备调换即可)
func MigrateIPAddrs(src, dst netlink.Link) (err error) {
	// 获取物理网卡的名称
	srcName := src.Attrs().Name
	// 获取网桥设备的名称
	dstName := dst.Attrs().Name

	// 获取指定设备上的相关路由, 这里是获取物理网卡的路由
	addrs, err := netlink.AddrList(src, netlink.FAMILY_V4)
	if err != nil {
		klog.Errorf("failed to get addresses of %s: %s.", srcName, err)
		return
	}

	// 打印获取到的 IP 地址
	klog.V(3).Infof("get addresses of device %s, len: %d: %+v", srcName, len(addrs), addrs)

	// 从 src 迁移 IP 时, 相关的路由就会同时被删除, 所以这里需要先获取, 之后要由 dst 设备接管 src 设备上的所有路由
	routes, err := netlink.RouteList(src, netlink.FAMILY_V4)
	if err != nil {
		klog.Errorf("failed to get routes of %s: %s.", srcName, err)
		return
	}

	// 打印获取到的路由
	klog.V(3).Infof("get routes of device %s, len: %d: %+v", srcName, len(routes), routes)

	// 迁移 IP 地址
	for _, addr := range addrs {
		// 从一个接口上移除 IP 地址, 对应的路由也会被移除
		if err = netlink.AddrDel(src, &addr); err != nil {
			klog.Errorf("failed to delete address on %s: %s.", srcName, err)
			continue
		}

		// 为接口添加一个 Label, 否则会出现 label must begin with interface name 的错误, 原因未知
		// 在 AddrAdd() 源码中有检验 Label 是否以接口名称为前缀的判断
		addr.Label = dstName

		// 添加 IP 地址到另一个接口上
		if err = netlink.AddrAdd(dst, &addr); err != nil {
			klog.Errorf("failed to add address to %s: %s.", dstName, err)
			return
		}
	}

	klog.V(3).Infof("move addresses from %s to %s successfully.", srcName, dstName)

	// 迁移路由
	return ModifyRoutes(routes, dst.Attrs().Index)
}

// ModifyRoutes IP 地址从物理网卡迁移到桥接设备后还需要修改相关的路由字段
func ModifyRoutes(routes []netlink.Route, devIndex int) (err error) {
	// 修改路由, 这里走逆向遍历, 因为 routes[0] 是默认路由, 在目标路由不指定的情况下添加默认路由会失败。
	// 所以我们需要先修改指定目标网络的路由

	// 遍历所有路由
	rlength := len(routes)
	for i := 0; i < rlength; i++ {
		// 逆向遍历
		route := routes[rlength-i-1]
		if err = netlink.RouteDel(&route); err != nil {
			// 有可能在移除物理网卡的 IP 时, 对应的路由就自动被移除了, 所以这里出错的话不 return
			if err.Error() != "no such process" {
				klog.Errorf("failed to delete route %+v: %s.", route, err)
				return
			}
		}

		// 变更路由主要是将路由条目的 dev 字段修改为网桥设备的索引
		route.LinkIndex = devIndex
		// 添加路由
		if err = netlink.RouteAdd(&route); err != nil {
			if err.Error() != "file exists" {
				klog.Errorf("failed to add route %+v: %s.", route, err)
				return
			}
		}
	}

	return
}

// GetBridgeAndEth0 获取网桥设备和物理网卡设备, 如果不存在则创建
func GetBridgeAndEth0(bridgeName, eth0Name string) (bridge netlink.Link, eth0 netlink.Link, err error) {
	// 获取网桥设备
	bridge, err = GetBridgeDevice(bridgeName)
	if err != nil {
		return
	}

	eth0, err = netlink.LinkByName(eth0Name)
	if err != nil {
		klog.Warningf("failed to get target device %s: %s", eth0Name, err)
		return
	}

	return
}

// InstallBridgeNetwork 部署桥接网络
// 手动创建 mybr0 接口, 然后将宿主机的主网卡 eth0 接入
// 因为如果不完成接入, invoke 调用 bridge + dhcp 插件时请求会失败
func InstallBridgeNetwork(bridgeName, eth0Name string) (err error) {
	// 调用 GetBridgeAntEth0() 函数获取网桥设备和物理网卡设备
	linkBridge, linkEth0, err := GetBridgeAndEth0(bridgeName, eth0Name)
	if err != nil {
		return
	}

	// 将 eth0 接入网桥设备
	if err = netlink.LinkSetMaster(linkEth0, linkBridge); err != nil {
		klog.Errorf("failed to set %s master to %s: %s", eth0Name, bridgeName, err)
		return err
	}

	klog.V(3).Infof("set %s master to %s successfully.", eth0Name, bridgeName)

	// 迁移 IP 地址
	if err = MigrateIPAddrs(linkEth0, linkBridge); err != nil {
		return err
	}

	return
}

// UninstallBridgeNetwork 卸载桥接网络
// 将物理网卡 eth0 从 mybr0 网桥设备中拔出, 并且恢复其路由配置
// 最终移除 mybr0 网桥设备
func UninstallBridgeNetwork(bridgeName, eth0Name string) (err error) {
	// 调用 GetBridgeAntEth0() 函数获取网桥设备和物理网卡设备
	linkBridge, linkEth0, err := GetBridgeAndEth0(bridgeName, eth0Name)
	if err != nil {
		return
	}

	// 将 eth0 从网桥设备中拔出
	if err = netlink.LinkSetNoMaster(linkEth0); err != nil {
		klog.Errorf("failed to set no master for %s: %s", eth0Name, err)
		return
	}

	klog.V(3).Infof("set no master for %s successfully.", eth0Name)

	// 迁移 IP 地址
	if err = MigrateIPAddrs(linkBridge, linkEth0); err != nil {
		return err
	}

	// 移除网桥设备
	if err = netlink.LinkDel(linkBridge); err != nil {
		klog.Errorf("failed to remove bridge device %s: %s", bridgeName, err)
		return
	}

	return
}

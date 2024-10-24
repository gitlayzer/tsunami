package podroute

import (
	"fmt"
	"net"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/gitlayzer/tsunami/pkg/cninet"
	"github.com/vishvananda/netlink"
	"k8s.io/klog"
)

// MakeServiceCIDRRoute 生成 Pod 到 ServiceIP 的路由
func MakeServiceCIDRRoute(linkBridge netlink.Link, serviceCIDR string) (svcRoute *netlink.Route, err error) {
	bridgeAddrs, err := netlink.AddrList(linkBridge, netlink.FAMILY_V4)
	if err != nil {
		return nil, fmt.Errorf("failed to get bridge address: %v", err)
	}

	klog.V(3).Infof("bridge addresses: %+v, len: %d", bridgeAddrs, len(bridgeAddrs))

	var gw net.IP
	if len(bridgeAddrs) > 0 {
		gw = bridgeAddrs[0].IP
	}

	// 创建路由规则，目的地址为 10.96.0.0/12，网关为 bridge 网卡的 IP 地址
	svcRoute = &netlink.Route{
		Dst: &net.IPNet{
			IP: net.IPv4(10, 96, 0, 0), Mask: net.CIDRMask(12, 32),
		},
		Gw: gw,
	}

	if serviceCIDR != "" {
		_, svcNet, err := net.ParseCIDR(serviceCIDR)
		if err != nil {
			return nil, fmt.Errorf("failed to parse service CIDR: %v", err)
		}
		svcRoute.Dst = svcNet
	}

	return
}

// SetRouteInPod 在 Pod 命名空间中设置路由规则，有两种情况
// 1：默认路由, 一般 bridge + dhcp 会自动为 Pod 创建默认路由, 在 ESXI 环境下, 创建的 Pod 申请到 IP 后并不会创建, 后续可能需要适配
// 2：Pod 到 ServiceIP 的路由, 需要设置宿主机为该 Pod 的网关, 否则拥有宿主机网络 IP 的 Pod 无法访问到 ServiceIP
func SetRouteInPod(bridgeName, netnsPath, serviceIPCIDR string) (svcRoute *netlink.Route, err error) {
	linkBridge, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return nil, fmt.Errorf("faliled to get bridge link: %s", err)
	}

	// 获取宿主机上的默认路由, 之后需要在设置容器中默认路由时使用ta的网关.
	hostDefRoute, err := cninet.GetDefaultRoute()
	if err != nil {
		hostDefRoute = nil
		klog.Warning(err)
	}

	svcRoute, err = MakeServiceCIDRRoute(linkBridge, serviceIPCIDR)
	if err != nil {
		return nil, fmt.Errorf("faliled to generate service route: %s", err)
	}

	netns, err := ns.GetNS(netnsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open netns %q: %v", netnsPath, err)
	}
	defer netns.Close()

	err = netns.Do(func(containerNS ns.NetNS) (err error) {
		link, err := netlink.LinkByName("eth0")
		if err != nil {
			return fmt.Errorf("faliled to get eth0 link: %s", err)
		}
		
		// 判断容器中是否存在默认路由, 如果不存在则创建(需要使用宿主机的网关).
		_, err = cninet.GetDefaultRoute()
		if err != nil {
			klog.Warning(err)
			defRoute := cninet.MakeDefaultRoute(hostDefRoute.Gw)
			defRoute.LinkIndex = link.Attrs().Index
			err = netlink.RouteAdd(defRoute)
			if err != nil {
				return fmt.Errorf("faliled to add default route: %s", err)
			}
		}
		// 添加到service cidr的路由.
		svcRoute.LinkIndex = link.Attrs().Index
		err = netlink.RouteAdd(svcRoute)
		if err != nil {
			return fmt.Errorf("faliled to add service cidr route: %s", err)
		}
		return nil
	})

	return svcRoute, err
}
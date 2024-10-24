package cninet

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

// GetDefaultRoute 获取默认路由
// 判断依据是 route 对象是否拥有 gw 成员，因为一般的路由只有 dst 成员，而没有 gw 成员。
// 如果没有默认路由，则返回 nil。
func GetDefaultRoute() (route *netlink.Route, err error) {
	// 获取默认路由, 这里只获取 IPv4 路由
	routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		return nil, fmt.Errorf("failed to get default route: %s", err)
	}

	for _, r := range routes {
		if r.Gw != nil {
			return &r, nil
		}
	}

	return nil, fmt.Errorf("default route doesn`t exist")
}

// MakeDefaultRoute 生成用于默认路由的对象，需要指定网关, 返回一个 netlink.Route 对象。
func MakeDefaultRoute(gw net.IP) *netlink.Route {
	// 构造默认网段
	_, defaultNet, _ := net.ParseCIDR("0.0.0.0/0")
	return &netlink.Route{
		// 全局路由
		Scope: netlink.SCOPE_UNIVERSE,
		// 目的地址
		Dst: defaultNet,
		// 网关
		Gw: gw,
	}
}

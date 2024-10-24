package main

import (
	"context"
	"encoding/json"
	"net"

	"github.com/containernetworking/cni/pkg/invoke"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/gitlayzer/tsunami/pkg/config"
	"github.com/gitlayzer/tsunami/pkg/podroute"
	"github.com/gitlayzer/tsunami/utils/restapi"
	"github.com/gitlayzer/tsunami/utils/skelargs"
	"github.com/gitlayzer/tsunami/utils/utilfile"
	"k8s.io/klog"
)

var (
	ver        = "0.3.1"
	versionAll = version.PluginSupports(ver)
)

// cmdAdd: 在调用此函数时, 以由kubelet创建好pause容器, 正是需要为其部署网络的时候.
// 而对应的业务容器此时还未创建.
func cmdAdd(args *skel.CmdArgs) (err error) {
	klog.Infof("cmdAdd args: %+v", args)
	netConf := &config.NetConf{}
	err = json.Unmarshal(args.StdinData, netConf)
	if err != nil {
		return
	}
	delegateBytes, err := json.Marshal(netConf.Delegate)
	if err != nil {
		return
	}

	// cni插件创建的, 默认的网络设备(名称一般为cni0).
	cni0 := netConf.Delegate["bridge"].(string)
	var resp *restapi.PodResponse
	var result types.Result
	podName, err := skelargs.ParseValueFromArgs("K8S_POD_NAME", args.Args)
	if err != nil {
		return err
	}
	podNS, err := skelargs.ParseValueFromArgs("K8S_POD_NAMESPACE", args.Args)
	if err != nil {
		return err
	}

	// 先判断 cniserver 进程是否存在.
	if utilfile.Exists(netConf.ServerSocket) {
		client := restapi.NewCNIServerClient(netConf.ServerSocket)
		resp, err = client.Add(&restapi.PodRequest{
			PodName:      podName,
			PodNamespace: podNS,
			ContainerID:  args.ContainerID,
			NetNs:        args.Netns,
			CNI0:         cni0,
		})

		if err != nil {
			klog.Errorf("failed to set network for pod: %s", err)
			return err
		}
	}

	// if 条件满足说明当前的Pod的确设置了静态IP, 需要为其生成 result 结果.
	if resp != nil {
		_, podIP, _ := net.ParseCIDR(resp.IPAddress)
		_, defnet, _ := net.ParseCIDR("0.0.0.0/0")
		gatewayIP := net.ParseIP(resp.Gateway).To4()
		result = &current.Result{
			CNIVersion: ver,
			IPs: []*current.IPConfig{
				{
					Version: "4",
					Address: *podIP,
					Gateway: gatewayIP,
				},
			},
			Routes: []*types.Route{
				{
					Dst: *defnet,
					GW:  gatewayIP,
				},
			},
		}
	} else {
		ipamType := netConf.Delegate["type"].(string)
		result, err = invoke.DelegateAdd(context.TODO(), ipamType, delegateBytes, nil)

		if err != nil {
			klog.Errorf("faliled to run bridge plugin: %s", err)
			return err
		}

		klog.Infof("run bridge plugin success: %s", result.String())
	}

	// 为 Pod 获取IP后, 检测是否存在默认路由, 并且添加Pod到ServiceCIRD的路由.
	_, err = podroute.SetRouteInPod(cni0, args.Netns, netConf.ServiceIPCIDR)
	if err != nil {
		klog.Errorf("faliled to add route to the pod %s: %s", args.Args, err)
		return
	}
	return result.Print()
}

func cmdDel(args *skel.CmdArgs) error {
	return nil
}

func cmdCheck(args *skel.CmdArgs) error {
	return nil
}

func main() {
	klog.Info("start cni-terway plugin...")
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, versionAll, "cni-terway")
}

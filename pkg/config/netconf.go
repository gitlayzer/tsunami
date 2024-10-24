package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/containernetworking/cni/pkg/types"
	"github.com/gitlayzer/tsunami/pkg/svcipcidr"
	"k8s.io/klog"
)

type NetConf struct {
	types.NetConf
	ServiceIPCIDR string `json:"serviceIPCIDR"`
	Delegate      map[string]interface{}
	// ServerSocket cni server 的 socket 路径
	// cni server 是用来设置容器内部为固定 IP 的
	ServerSocket string `json:"server_socket"`
}

// Complete 从 apiserver 获取 service cidr 范围, 然后写入到 cni netconf 中
func (n *NetConf) Complete(netConfPath string) (err error) {
	// 读取配置文件
	netConfContent, err := os.ReadFile(netConfPath)
	if err != nil {
		return fmt.Errorf("failed to read netconf file: %v", err)
	}

	// 解析配置文件到 NetConf 结构体
	if err = json.Unmarshal(netConfContent, n); err != nil {
		return fmt.Errorf("failed to parse netconf file: %v", err)
	}

	// 从 apiserver 获取 service cidr 范围
	serviceIPCIDR, err := svcipcidr.GetServiceIPCIDR()
	if err != nil {
		return fmt.Errorf("failed to get service IP CIDR: %v", err)
	}
	klog.Infof("get service ip cidr: %s", serviceIPCIDR)

	// 写入到 NetConf 结构体中
	n.ServiceIPCIDR = serviceIPCIDR

	// 重新写入配置文件
	netConfContent, err = json.Marshal(n)
	if err != nil {
		return fmt.Errorf("failed to marshal netconf: %v", err)
	}

	// 写入配置文件
	if err = os.WriteFile(netConfPath, netConfContent, 0644); err != nil {
		return fmt.Errorf("failed to write into netconf file: %v", err)
	}

	return
}

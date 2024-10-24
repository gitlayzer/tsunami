package svcipcidr

import (
	"context"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

// GetServiceIPCIDR 从 apiserver 组件对象中获取 service IP CIDR
func GetServiceIPCIDR() (serviceIPCIDR string, err error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		klog.Errorf("failed to get in cluster config: %v", err)
		return "", err
	}

	// 创建 clientset 客户端
	client, err := clientset.NewForConfig(cfg)
	if err != nil {
		klog.Errorf("failed to create clientset: %v", err)
		return "", err
	}

	// 设置一个 获取 kube-apiserver pod 的 label
	labelSet := labels.Set{
		"component": "kube-apiserver",
	}

	// 使用 labelSelector 获取 kube-apiserver Pod
	podListOpts := metav1.ListOptions{
		LabelSelector: labelSet.String(),
	}

	// 获取 kube-apiserver Pod
	podList, err := client.CoreV1().Pods("kube-system").List(context.Background(), podListOpts)
	if err != nil {
		klog.Errorf("failed to list pods: %v", err)
		return "", err
	}

	// 遍历 kube-apiserver Pod 的命令行参数，获取 service IP CIDR
	pod := podList.Items[0]
	cmds := pod.Spec.Containers[0].Command
	for _, cmd := range cmds {
		if strings.Contains(cmd, "service-cluster-ip-range") {
			// 获取 service IP CIDR 后, 退出循环
			serviceIPCIDR = strings.Split(cmd, "=")[1]
			break
		}
	}

	return
}

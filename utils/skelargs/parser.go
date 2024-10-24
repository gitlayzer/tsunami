package skelargs

import (
	"fmt"
	"strings"
)

// ParseValueFromArgs 从 cmdAdd/cmdDel 传入的 skel.CmdArgs 参数中
func ParseValueFromArgs(key, argStr string) (string, error) {
	// 检查 argStr 是否为空。如果为空，返回一个空字符串和错误信息，提示 CNI 参数是必需的。
	if argStr == "" {
		return "", fmt.Errorf("cni args is required")
	}

	// 使用 strings.Split 函数将 argStr 按分号 ; 分割，生成一个字符串切片 args，每个元素都是一个键值对。
	args := strings.Split(argStr, ";")

	// 构建一个字符串 keyPrefix，其值为 "key="，用于后续查找，以提高性能，避免在循环中重复计算。
	keyPrefix := fmt.Sprintf("%s=", key)

	// 开始一个循环，遍历切片 args 中的每个元素 arg。
	for _, arg := range args {
		// 检查当前的 arg 是否以 keyPrefix 开头。即检查是否找到了目标字段。
		if strings.HasPrefix(arg, keyPrefix) {
			// 如果找到了匹配的字段，使用 strings.TrimPrefix 函数去掉前缀 keyPrefix，返回去掉前缀后的值和 nil（表示没有错误）。
			return strings.TrimPrefix(arg, keyPrefix), nil
		}
	}

	// 如果遍历结束仍未找到指定的 key，返回一个空字符串和错误信息，提示该字段在 CNI 参数中是必需的。
	return "", fmt.Errorf("%s is required in cni args", key)
}

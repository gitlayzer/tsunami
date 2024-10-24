package utilfile

import "os"

// Exists 判断给到的路径文件 / 目录是否存在, 返回 bool 值： true 存在， false 不存在
func Exists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}

	return true
}

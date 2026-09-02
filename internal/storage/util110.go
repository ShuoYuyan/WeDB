package storage

import "os"

// fileExists 供加密层与 Pager 判断附属文件是否存在
func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

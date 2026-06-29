//go:build !windows

package main

import (
	"fmt"
)

func SetDllPath(path string) error {
	fmt.Printf("DLL 路径设置仅在 Windows 平台支持\n")
	return nil
}

func GetDefaultDllPath() string {
	return ""
}

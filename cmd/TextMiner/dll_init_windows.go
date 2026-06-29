//go:build windows

package main

/*
#cgo windows LDFLAGS: -lkernel32

#include <windows.h>
#include <stdlib.h>

static void SetDllDirectoryPath(const char* path) {
    int len = MultiByteToWideChar(CP_UTF8, 0, path, -1, NULL, 0);
    if (len > 0) {
        wchar_t* wpath = (wchar_t*)malloc(len * sizeof(wchar_t));
        if (wpath) {
            MultiByteToWideChar(CP_UTF8, 0, path, -1, wpath, len);
            SetDllDirectoryW(wpath);
            free(wpath);
        }
    }
}

static void AddDllDirectoryPath(const char* path) {
    int len = MultiByteToWideChar(CP_UTF8, 0, path, -1, NULL, 0);
    if (len > 0) {
        wchar_t* wpath = (wchar_t*)malloc(len * sizeof(wchar_t));
        if (wpath) {
            MultiByteToWideChar(CP_UTF8, 0, path, -1, wpath, len);
            AddDllDirectory(wpath);
            free(wpath);
        }
    }
}
*/
import "C"
import (
	"fmt"
	"os"
	"path/filepath"
	"unsafe"
)

func SetDllPath(path string) error {
	if path == "" {
		return fmt.Errorf("DLL 路径为空")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("DLL 目录不存在: %s", path)
	}

	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	C.SetDllDirectoryPath(cPath)
	C.AddDllDirectoryPath(cPath)

	return nil
}

func GetDefaultDllPath() string {
	execPath, err := os.Executable()
	if err != nil {
		return ""
	}

	execDir := filepath.Dir(execPath)

	if is64Bit() {
		return filepath.Join(execDir, "lib", "x64")
	} else {
		return filepath.Join(execDir, "lib", "x86")
	}
}

func is64Bit() bool {
	return unsafe.Sizeof(0) == 8
}

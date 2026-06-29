package main

/*
#include <stdlib.h>
#include <string.h>
#include <windows.h>

static char* alloc_string(const char* str) {
    size_t len = strlen(str) + 1;
    char* result = (char*)malloc(len);
    if (result) {
        memcpy(result, str, len);
    }
    return result;
}

static char* get_dll_directory() {
    char path[MAX_PATH];
    HMODULE hModule = NULL;

    if (GetModuleHandleExA(GET_MODULE_HANDLE_EX_FLAG_FROM_ADDRESS, (void*)get_dll_directory, &hModule)) {
        GetModuleFileNameA(hModule, path, MAX_PATH);
        char* lastSlash = strrchr(path, '\\');
        if (lastSlash) {
            *lastSlash = '\0';
        }
        return alloc_string(path);
    }

    return alloc_string(".");
}

static void load_onnx_runtime() {
    char path[MAX_PATH];
    HMODULE hModule = NULL;

    if (GetModuleHandleExA(GET_MODULE_HANDLE_EX_FLAG_FROM_ADDRESS, (void*)load_onnx_runtime, &hModule)) {
        GetModuleFileNameA(hModule, path, MAX_PATH);
        char* lastSlash = strrchr(path, '\\');
        if (lastSlash) {
            *lastSlash = '\0';
            strcat(path, "\\onnxruntime.dll");
            LoadLibraryA(path);
        }
    }
}
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"unsafe"

	"textminer/pkg/extractor"
)

type ExtractResultC struct {
	FileName     string `json:"file_name"`
	FileType     string `json:"file_type"`
	FileSize     int64  `json:"file_size"`
	Status       string `json:"status"`
	Content      string `json:"content"`
	ErrorMessage string `json:"error_message"`
}

func init() {
	C.load_onnx_runtime()
}

//export TextMiner_ExtractFile
func TextMiner_ExtractFile(filePath *C.char, enableOcr C.int) *C.char {
	if filePath == nil {
		return C.CString(`{"status":"failed","error_message":"file path is nil"}`)
	}
	goFilePath := C.GoString(filePath)
	goEnableOcr := enableOcr != 0

	result, err := extractor.ExtractFile(goFilePath, goEnableOcr)
	if err != nil {
		fmt.Printf("ExtractFile failed: %v\n", err)
		resultC := ExtractResultC{
			Status:       extractor.StatusFailed,
			ErrorMessage: fmt.Sprintf("提取文件失败: %v", err),
		}
		jsonData, _ := json.Marshal(resultC)
		return C.CString(string(jsonData))
	}

	resultC := ExtractResultC{
		FileName:     result.FileName,
		FileType:     result.FileType,
		FileSize:     result.FileSize,
		Status:       result.Status,
		Content:      result.Content,
		ErrorMessage: result.ErrorMessage,
	}

	jsonData, err := json.Marshal(resultC)
	if err != nil {
		fmt.Printf("JSON marshal failed: %v\n", err)
		return C.CString(`{"status":"failed","error_message":"JSON序列化失败"}`)
	}

	return C.CString(string(jsonData))
}

func main() {
}

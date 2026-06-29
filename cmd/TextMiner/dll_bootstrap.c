#include <windows.h>
#include <stdlib.h>
#include <stdio.h>

// GCC/MinGW 的构造函数，在程序启动时自动执行
__attribute__((constructor))
static void set_dll_search_path_gcc() {
    char exe_path[MAX_PATH];
    GetModuleFileNameA(NULL, exe_path, MAX_PATH);
    
    // 获取可执行文件所在目录
    char* last_backslash = strrchr(exe_path, '\\');
    if (last_backslash) {
        *last_backslash = '\0';
    }
    
    // 构建默认的 lib 路径
    char lib_path[MAX_PATH];
    snprintf(lib_path, MAX_PATH, "%s\\lib", exe_path);
    
    // 检查 lib 目录是否存在
    DWORD attrs = GetFileAttributesA(lib_path);
    if (attrs != INVALID_FILE_ATTRIBUTES && (attrs & FILE_ATTRIBUTE_DIRECTORY)) {
        // lib 目录存在，设置为 DLL 搜索路径
        SetDllDirectoryA(lib_path);
    }
}

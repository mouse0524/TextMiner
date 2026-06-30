# TextMiner 文件内容提取工具

## 简介
TextMiner 文件内容提取工具，支持多种文件类型的文本内容提取，包括文本文件、Office文件、PDF文件和代码文件。

## 支持的文件类型

### Office 文件
- **Word 文档**：doc, docx, dot, dotx, dotm, docm
- **PowerPoint 演示文稿**：ppt, pptx, pot, potx, potm, pps, ppsx, ppsm, pptm
- **Excel 表格**：xls, xlsx, xlsm, xlt, xltx, xltm
- **Excel 加载宏**：xlam
- **WPS Office**：wps, wpt, dps, dpt, et, ett

### PDF 文件
- **PDF 文档**：pdf（支持 OCR 识别扫描文档）

### 文本文件
- **纯文本**：txt（支持 UTF-8、GBK、GB18030、UTF-16 等多种编码）
- **日志文件**：log
- **CSV 文件**：csv（支持自动识别分隔符）
- **配置文件**：ini
- **RTF 文档**：rtf（富文本格式）

### 代码文件
- **Go**：go
- **Java**：java
- **Python**：py, pyw
- **C/C++**：c, cpp, h, hpp, cc, cxx, hh, hxx
- **JavaScript/TypeScript**：js, jsx, ts, tsx
- **Web 前端**：html, htm, css, scss, sass, less
- **PHP**：php
- **Rust**：rs
- **Swift**：swift
- **Kotlin**：kt, kts
- **Scala**：scala
- **Ruby**：rb
- **Perl**：pl, pm
- **SQL**：sql
- **配置文件**：xml, json, yaml, yml
- **文档**：md, markdown
- **脚本**：sh, bash, bat, ps1

### 压缩包文件
- **ZIP**：zip
- **7-Zip**：7z
- **RAR**：rar
- **TAR**：tar, tar.gz, tgz, gz

### 图片文件
- **常见图片格式**：png, jpg, jpeg, bmp, gif, tiff, tif（支持 OCR 识别）

### 音频文件
- **常见音频格式**：mid, midi, wav, ogg, oga, ogx, mp3, 8svx, aac, ac3, aiff, aif, amb, amr, au, avr, caf, cdda, cvs, cvsd, cvu, dts, dvms, fap, flac, fssd, gsrt, hcom, htk, ima, ircam, m4a, m4b, m4p, m4r, maud, mmf, mp2, nist, opus, paf, pcma, pcmu, prc, pvf, ra, ram, sd2, sln, smp, snd, sndr, sndt, sou, sph, spx, tta, txw, vms, voc, vox, w64, wma, wv, wve

### 视频文件
- **常见视频格式**：swf, mp4, mpg, wmv, 3g2, 3gp, asf, avi, dat, dv, f4v, flv, hevc, m2ts, m2v, m4v, mjpeg, mkv, mov, mpeg, mts, mxf, ogv, rm, rmvb, vob, webm, wtv

### 其他文件类型
- **Android 应用**：apk
- **电子书**：azw3, lrf
- **3D 模型**：blend, c4d, fbx, max, stl, x3d
- **CAD 文件**：catpart, dwg, dws, dxf, prt, sldasm, sldprt
- **文档格式**：chm, djvu, mht, mhtml, snb, xps
- **数据库**：dbf
- **医学影像**：dcm
- **音频播放列表**：m3u, m3u8
- **邮件**：eml, vcf
- **可执行文件**：exe
- **压缩包**：bz2, xz, tar.bz2, tar.xz, rpm, iso
- **其他**：daf, dsm, in, jar, tex, xpi

## 项目结构

```
textminer/
├── cmd/
│   ├── TextMiner/                    # 命令行工具入口
│   │   ├── main.go             # 主程序
│   │   ├── TextMiner.exe.manifest    # 应用程序清单
│   │   ├── dll_bootstrap.c     # DLL引导代码
│   │   ├── dll_init_unix.go    # Unix平台DLL初始化
│   │   └── dll_init_windows.go # Windows平台DLL初始化
│   └── TextMiner.dll/                # DLL入口
│       └── main.go             # DLL主程序
├── pkg/
│   ├── extractor/              # 核心提取功能包
│   │   ├── extractor.go        # 核心接口定义
│   │   ├── txt_extractor.go    # 文本文件提取器
│   │   ├── pdf_extractor.go    # PDF文件提取器
│   │   ├── office_extractor.go # Office文件提取器
│   │   ├── office_embedding_extractor.go # Office嵌入文件提取器
│   │   ├── code_extractor.go   # 代码文件提取器
│   │   ├── image_extractor.go  # 图片文件提取器
│   │   ├── audio_extractor.go  # 音频文件提取器
│   │   ├── video_extractor.go  # 视频文件提取器
│   │   ├── mime_only_extractor.go # MIME类型识别器
│   │   ├── archive_extractor.go # 压缩包提取器
│   │   ├── csv_extractor.go    # CSV文件提取器
│   │   ├── rtf_extractor.go    # RTF文件提取器
│   │   ├── legacy_extractor.go  # 遗留文件提取器
│   │   ├── xlsb_parser.go      # XLSB文件解析器
│   │   ├── ocr_processor.go    # OCR处理器
│   │   ├── encryption_detector.go # 加密检测器
│   │   └── file_type_detector.go # 文件类型检测器
│   ├── logger/                 # 日志模块
│   │   └── logger.go           # 日志系统实现
│   ├── TextMiner/                    # DLL接口
│   │   └── dll_windows.go      # Windows DLL实现
│   └── magika/                 # 文件类型检测（基于ONNX Runtime）
│       ├── magika/             # Magika核心
│       │   ├── config.go       # 配置
│       │   ├── content.go      # 内容处理
│       │   ├── detector.go     # 检测器
│       │   ├── errors.go       # 错误定义
│       │   ├── features.go     # 特征提取
│       │   └── scanner.go      # 扫描器
│       └── onnx/              # ONNX Runtime封装
│           ├── onnx.go         # ONNX接口
│           ├── onnx_runtime.h  # ONNX Runtime头文件
│           ├── onnxruntime_ep_c_api.h # ONNX Runtime扩展API
│           ├── onnx_runtime_unix.go   # Unix平台实现
│           ├── onnx_runtime_windows.go # Windows平台实现
│           └── onnx_zero.go    # ONNX零依赖实现
├── local/                      # 本地依赖（通过 go.mod replace 指令引用）
│   ├── onnxruntime_purego/    # 纯Go ONNX Runtime封装
│   │   ├── api.go              # API接口
│   │   ├── onnxruntime.go      # ONNX Runtime实现
│   │   ├── sesstion.go         # 会话管理
│   │   ├── utils.go            # 工具函数
│   │   ├── utils_unix.go       # Unix工具
│   │   ├── utils_windows.go    # Windows工具
│   │   ├── value.go            # 值处理
│   │   └── ...
│   ├── go-catdoc-main/        # 旧版 Word 文档（.doc）处理
│   │   ├── catdoc.go
│   │   ├── charsets/           # 字符集定义
│   │   └── ...
│   ├── goppt-main/            # 旧版 PowerPoint 文档（.ppt）处理
│   │   ├── ppt.go
│   │   ├── record.go
│   │   └── internal/ioadapters/
│   └── xlsReader-master/      # 旧版 Excel 文档（.xls）读取
│       ├── cfb/                # Compound File Binary 解析
│       ├── helpers/
│       └── xls/                # XLS 记录解析
├── models/                     # ONNX模型文件
│   ├── config.min.json         # 模型配置
│   ├── content_types_kb.min.json # 内容类型知识库
│   ├── det.onnx               # 检测模型
│   ├── dict.txt               # 字典文件
│   ├── metadata.json          # 元数据
│   ├── model.onnx             # 主模型
│   └── rec.onnx               # 识别模型
├── lib/                       # 依赖库目录
│   ├── x86/                  # 32位依赖库
│   │   ├── onnxruntime.dll
│   │   ├── onnxruntime_providers_shared.dll
│   │   ├── fastkn32.dll
│   │   ├── fastonnx.dll
│   │   └── ...               # 其他依赖库
│   └── x64/                  # 64位依赖库
│       ├── onnxruntime.dll
│       ├── onnxruntime_providers_shared.dll
│       ├── fastkn64.dll
│       ├── fastonnx.dll
│       └── ...               # 其他依赖库
├── examples/                   # 示例代码
│   └── cpp/                  # C++示例
│       └── simple.cpp         # 简单示例
├── build-all.bat               # 统一构建脚本（x86+x64）
├── build-dll.bat              # DLL构建脚本（x86+x64）
├── go.mod                      # 依赖管理
├── go.sum                      # 依赖校验
├── .gitignore                  # Git忽略文件
├── LICENSE                     # 许可证
└── README.md                   # 项目说明
```

## 安装与使用

### 环境要求

- Go 1.24.4+（与 `go.mod` 中声明的工具链版本一致）
- Windows 7 SP1 及以上版本
- CGO 编译器（MinGW32 / MinGW64，用于 `onnxruntime` C 绑定）
- 若使用 `build-all.bat` / `build-dll.bat`，需将 `mingw32`、`mingw64` 目录放置于项目根目录

### 安装依赖

```bash
go mod download
```

### 构建命令行工具

#### 使用统一构建脚本（推荐）

```bash
# 构建所有版本（x86 + x64）
.\build-all.bat
```

### 运行程序

**重要：运行前需要设置 PATH 环境变量**

#### 方法一：临时设置 PATH（推荐用于测试）

**PowerShell:**
```powershell
# 设置 32 位版本 PATH
$env:PATH = ".\build\x86\lib;$env:PATH"
.\build\x86\TextMiner.exe <文件路径>

# 设置 64 位版本 PATH
$env:PATH = ".\build\x64\lib;$env:PATH"
.\build\x64\TextMiner.exe <文件路径>
```

**CMD:**
```cmd
REM 设置 32 位版本 PATH
set PATH=.\build\x86\lib;%PATH%
build\x86\TextMiner.exe <文件路径>

REM 设置 64 位版本 PATH
set PATH=.\build\x64\lib;%PATH%
build\x64\TextMiner.exe <文件路径>
```


#### 方法二：永久设置系统环境变量

1. 右键"此电脑" → "属性" → "高级系统设置"
2. 点击"环境变量"
3. 在"用户变量"或"系统变量"中找到"Path"
4. 点击"编辑"，添加 lib 目录路径
5. 点击"确定"保存

### 使用方法

#### 命令格式

```text
TextMiner [flags] <文件路径>
TextMiner version
```

可用参数：

| 参数 | 说明 | 默认值 |
| --- | --- | --- |
| `--ocr` | 启用 OCR 识别（对扫描型 PDF 与图片有效） | `false` |
| `--output` | 将提取内容写入同名 `.txt` 文件 | `false` |
| `version` | 子命令，打印工具版本 | — |

#### 基本用法

```bash
# 提取单个文件内容（结果以 JSON 格式输出到 stdout）
./TextMiner.exe <文件路径>
```

#### 查看版本信息

```bash
./TextMiner.exe version
```

#### 示例

```bash
# 提取 Word 文档内容
.\build\x86\TextMiner.exe test.docx

# 提取 PDF 文档内容（带 OCR）
.\build\x86\TextMiner.exe --ocr test.pdf

# 提取并输出到 txt 文件
.\build\x86\TextMiner.exe --output test.docx

# 同时启用 OCR 并写出 txt
.\build\x64\TextMiner.exe --ocr --output scan.pdf
```

#### 输出格式

执行成功后，程序会以如下 JSON 结构输出到 stdout：

```json
{
  "file_name": "test",
  "file_type": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
  "file_size": 12345,
  "status": "success",
  "content": "提取到的正文内容...",
  "error_message": ""
}
```

> 提示：完整运行日志位于 `%APPDATA%\iandsec\logs`，按天轮转，最多保留 30 天。

### 库函数调用（嵌入到其他 Go 项目）

您可以将此工具作为库集成到其他Go项目中：

```go
import (
    "fmt"
    "textminer/pkg/extractor"
)

func main() {
    filePath := "test.pdf"
    result, err := extractor.ExtractFile(filePath, false)
    if err != nil {
        fmt.Printf("提取失败: %v\n", err)
        return
    }
    fmt.Printf("文件名: %s\n", result.FileName)
    fmt.Printf("文件类型: %s\n", result.FileType)
    fmt.Printf("提取状态: %s\n", result.Status)
    fmt.Printf("提取内容: %s\n", result.Content)
}
```

## 构建 DLL（供 C++ 调用）

本项目支持编译为 Windows DLL，可供 C++ 等语言调用。

#### 构建 DLL

```bash
# 构建所有版本（x86 + x64）
.\build-dll.bat
```

构建完成后，生成的文件位于 `build-dll` 目录：

```
build-dll/
├── x86/
│   ├── TextMiner.dll        # 32位 DLL
│   ├── TextMiner.h          # C 头文件（自动生成）
│   ├── models/         # Magika 模型文件
│   └── *.dll          # 依赖库（onnxruntime.dll 等）
└── x64/
    ├── TextMiner.dll        # 64位 DLL
    ├── TextMiner.h          # C 头文件（自动生成）
    ├── models/         # Magika 模型文件
    └── *.dll          # 依赖库（onnxruntime.dll 等）
```

#### DLL 导出函数

DLL 导出以下函数：

- `TextMiner_ExtractFile(filePath *C.char, enableOcr C.int) *C.char` - 提取文件内容，返回 JSON 格式字符串

**注意**：返回的字符串由 `C.CString` 分配，落在 C 堆上。调用方在使用完毕后必须使用与 `malloc` 配对的方式释放：

- C / C++：`free(result)`，需 `extern "C" { extern void free(void*); }` 或 `<cstdlib>`。
- Python `ctypes`：`ctypes.free(result_ptr)`。
- C# `Marshal.FreeHGlobal` / `Marshal.PtrToStringAnsi` 之后调用 `FreeHGlobal`。

直接丢弃指针会导致进程内存持续增长；直接调用 `delete`（C++）会因分配器不匹配而崩溃。

#### C++ 调用示例

```cpp
#include <iostream>
#include <windows.h>

extern "C" {
    typedef char* (*TextMiner_ExtractFileFunc)(const char* filePath, int enableOcr);
}

int main() {
    HMODULE hDll = LoadLibraryA("textminer.dll");
    if (!hDll) {
        std::cerr << "Failed to load DLL" << std::endl;
        return 1;
    }

    TextMiner_ExtractFileFunc TextMiner_ExtractFile = 
        (TextMiner_ExtractFileFunc)GetProcAddress(hDll, "TextMiner_ExtractFile");
    
    if (!TextMiner_ExtractFile) {
        std::cerr << "Failed to get function address" << std::endl;
        FreeLibrary(hDll);
        return 1;
    }

    char* result = TextMiner_ExtractFile("test.docx", 0);
    
    std::cout << "Result: " << result << std::endl;
    
    FreeLibrary(hDll);
    return 0;
}
```

完整示例代码请参考 `examples/cpp/simple.cpp`。

#### Python 调用示例

> 注：完整的 Python 调用示例代码位于 `build-dll\x86\TextMiner_example.py` 和 `build-dll\x64\TextMiner_example.py`，
> 由 `build-dll.bat` 脚本在构建完成后从 `examples\python\` 拷贝生成（首次构建前需自行准备该目录及示例文件）。

```python
import ctypes
import json

dll = ctypes.CDLL("textminer.dll")

dll.TextMiner_ExtractFile.argtypes = [ctypes.c_char_p, ctypes.c_int]
dll.TextMiner_ExtractFile.restype = ctypes.c_char_p

file_path = "test.docx".encode('utf-8')
result_ptr = dll.TextMiner_ExtractFile(file_path, 0)

result_str = ctypes.string_at(result_ptr).decode('utf-8')
result = json.loads(result_str)

ctypes.free(result_ptr)

print(f"Status: {result['status']}")
print(f"Content: {result['content']}")
```

## 核心特性

1. **模块化设计**：采用接口设计，方便扩展支持更多文件类型
2. **自动编码检测**：支持多种文本编码，自动检测并解码
3. **完善的错误处理**：详细的错误信息，便于调试和使用
4. **统一的输出格式**：所有文件类型采用相同的JSON输出格式
5. **易于扩展**：新增文件类型只需实现Extractor接口
6. **命令行友好**：使用cobra库实现强大的命令行界面
7. **OCR 支持**：支持对扫描的PDF文档和图片进行OCR文字识别
8. **日志记录**：内置日志系统，支持按天轮转，最多保留30天日志
9. **DLL 路径管理**：通过 `dll_bootstrap.c` 引导可执行文件在启动时将所在目录的 `lib` 子目录加入 DLL 搜索路径（命令行程序同样适用）
10. **多架构支持**：同时提供 32 位和 64 位版本，兼容不同系统
11. **智能文件检测**：基于 Magika 的文件类型检测，准确识别未知文件类型
12. **性能优化**：针对 PPT 和 PPTX 文件进行了深度性能优化，提升大文件处理速度
13. **密码检测优化**：对特定文件类型（ppt, pot, pps, dps, dpt）不进行密码检测，提升提取效率
14. **音频文件支持**：支持48种音频格式的MIME类型识别
15. **视频文件支持**：支持27种视频格式的MIME类型识别
16. **扩展文件类型**：支持36种其他文件类型的MIME类型识别，包括APK、3D模型、CAD文件等
17. **MIME类型优先**：对于音频、视频等文件，优先使用扩展名进行MIME类型识别，避免Magika误判
18. **本地依赖管理**：使用本地依赖替换，确保特定版本的兼容性和稳定性
19. **压缩包支持**：支持ZIP、7Z、RAR、TAR、GZ、BZ2、XZ、ISO、RPM等多种压缩格式
20. **Office嵌入文件**：支持提取Office文档中的嵌入文件内容
21. **加密检测**：自动检测文件是否加密，避免尝试提取加密文件

## 构建输出

使用构建脚本后，会在 `build/` 目录下生成以下内容：

```
build/
├── x86/
│   ├── TextMiner.exe          # 32 位可执行文件
│   ├── lib/                 # 32 位依赖库
│   │   ├── onnxruntime.dll
│   │   ├── fastkn32.dll
│   │   ├── fastonnx.dll
│   │   └── ...
│   └── models/              # ONNX 模型文件
│       ├── config.min.json
│       ├── det.onnx
│       ├── rec.onnx
│       └── dict.txt
└── x64/
    ├── TextMiner.exe          # 64 位可执行文件
    ├── lib/                 # 64 位依赖库
    │   ├── onnxruntime.dll
    │   └── ...
    └── models/              # ONNX 模型文件
        └── ...
```

**注意**：
- 运行程序前需要将对应的 `lib` 目录添加到 PATH 环境变量
- models 文件会自动复制到 build 目录

## 依赖库

### 主要依赖
- [github.com/spf13/cobra](https://github.com/spf13/cobra) - 命令行框架
- [github.com/unidoc/unipdf/v3](https://github.com/unidoc/unipdf/v3) - PDF文件处理和渲染
- [github.com/unidoc/unioffice](https://github.com/unidoc/unioffice) - Office文件处理
- [github.com/getcharzp/go-ocr](https://github.com/getcharzp/go-ocr) - OCR文字识别
- [github.com/bodgit/sevenzip](https://github.com/bodgit/sevenzip) - 7z压缩包处理
- [github.com/nwaples/rardecode/v2](https://github.com/nwaples/rardecode/v2) - RAR压缩包处理
- [github.com/ulikunitz/xz](https://github.com/ulikunitz/xz) - XZ压缩格式处理
- [github.com/kdomanski/iso9660](https://github.com/kdomanski/iso9660) - ISO镜像文件处理
- [github.com/cavaliergopher/rpm](https://github.com/cavaliergopher/rpm) - RPM包文件处理
- [github.com/EndFirstCorp/peekingReader](https://github.com/EndFirstCorp/peekingReader) - 读取器工具
- [github.com/up-zero/gotool](https://github.com/up-zero/gotool) - 通用工具库
- [golang.org/x/text](https://golang.org/x/text) - 文本编码处理
- [golang.org/x/crypto](https://golang.org/x/crypto) - 加密相关功能
- [golang.org/x/image](https://golang.org/x/image) - 图像处理
- [golang.org/x/sys](https://golang.org/x/sys) - 系统调用封装

### 本地依赖
- [github.com/getcharzp/onnxruntime_purego](./local/onnxruntime_purego) - 纯Go ONNX Runtime封装
- [github.com/semvis123/go-catdoc](./local/go-catdoc-main) - 旧版Word文档处理
- [github.com/KSpaceer/goppt](./local/goppt-main) - 旧版PowerPoint文档处理
- [github.com/shakinm/xlsReader](./local/xlsReader-master) - XLS文件读取

### 间接依赖
- [github.com/sirupsen/logrus](https://github.com/sirupsen/logrus) - 日志系统
- [github.com/klauspost/compress](https://github.com/klauspost/compress) - 压缩算法库
- [github.com/pierrec/lz4/v4](https://github.com/pierrec/lz4/v4) - LZ4压缩算法
- [github.com/andybalholm/brotli](https://github.com/andybalholm/brotli) - Brotli压缩算法
- [github.com/tetratelabs/wazero](https://github.com/tetratelabs/wazero) - WebAssembly运行时
- [github.com/ebitengine/purego](https://github.com/ebitengine/purego) - CGO替代方案
- [github.com/richardlehane/mscfb](https://github.com/richardlehane/mscfb) - Microsoft Compound File Binary格式
- [github.com/richardlehane/msoleps](https://github.com/richardlehane/msoleps) - OLE属性集处理
- [github.com/unidoc/pkcs7](https://github.com/unidoc/pkcs7) - PKCS#7加密标准
- [github.com/unidoc/unitype](https://github.com/unidoc/unitype) - Unicode类型处理
- [github.com/bodgit/plumbing](https://github.com/bodgit/plumbing) - 压缩包底层工具
- [github.com/bodgit/windows](https://github.com/bodgit/windows) - Windows平台支持
- [github.com/hashicorp/golang-lru/v2](https://github.com/hashicorp/golang-lru/v2) - LRU缓存实现
- [github.com/hashicorp/errwrap](https://github.com/hashicorp/errwrap) - 错误包装
- [github.com/hashicorp/go-multierror](https://github.com/hashicorp/go-multierror) - 多错误处理
- [github.com/spf13/pflag](https://github.com/spf13/pflag) - 命令行参数解析
- [github.com/stretchr/testify](https://github.com/stretchr/testify) - 测试框架
- [github.com/davecgh/go-spew](https://github.com/davecgh/go-spew) - 深度打印工具
- [github.com/pmezard/go-difflib](https://github.com/pmezard/go-difflib) - 差异比较
- [github.com/inconshreveable/mousetrap](https://github.com/inconshreveable/mousetrap) - Windows控制台陷阱
- [github.com/metakeule/fmtdate](https://github.com/metakeule/fmtdate) - 日期格式化
- [gopkg.in/yaml.v3](https://gopkg.in/yaml.v3) - YAML解析
- [go4.org](https://go4.org) - Go4工具库

## 常见问题

### Q: 运行程序时报错"找不到 DLL"？
A:

- **命令行程序**（`TextMiner.exe`）：`build_helpers/dll_bootstrap.c` 会在启动时把可执行文件所在目录下的 `lib` 子目录加入 DLL 搜索路径，因此**只要保持构建产物布局**（`build\x86\TextMiner.exe` + `build\x86\lib\*.dll`）就能正常加载。如果手动移动，请同时把 `lib` 子目录搬到同一层。
- **DLL 模式**（`TextMiner.dll`）：依赖的 `onnxruntime.dll` 等必须与 DLL 在同一目录。
- **应急方案**：把对应 lib 目录加入 `PATH` 环境变量（详见"运行程序"章节）。

### Q: 支持哪些 Windows 版本？
A: 支持 Windows 7 SP1 及以上版本。

### Q: 如何选择 32 位还是 64 位版本？
A: 根据您的操作系统选择，64 位系统优先使用 x64 版本。

### Q: OCR 功能需要额外配置吗？
A: 不需要，所有模型文件已包含在 `models/` 目录中。只需确保 `onnxruntime.dll` 在 lib 目录下即可。

### Q: 构建时出现 `package textminer/pkg/xxx is not in std (xxx\src\textminer\pkg\xxx)` 错误？
A: 这是 Go 在 `GOPATH` 模式下找不到模块包的典型报错，常见原因与解决方案如下：

1. **未在项目根目录执行构建**：`go build` 必须在包含 `go.mod` 的目录（即项目根目录）下执行，`build-all.bat` / `build-dll.bat` 默认已切换到 `%CD%`，请勿在子目录中运行。
2. **Go 环境变量被修改**：执行 `go env GO111MODULE` 应输出 `on`。若为 `off`，执行 `go env -w GO111MODULE=on` 重新开启模块模式。
3. **自定义 GOROOT 下没有 std 目录**：本项目使用 `D:\go-legacy-win7` 作为 GOROOT，必须确保该目录下存在完整的 `src`、`pkg`、`bin` 子目录（`go env GOROOT` 可查看实际路径）。
4. **`go.mod` 缺失或被破坏**：确认项目根目录存在 `go.mod`，且包含 `module textminer` 一行；若丢失，从版本控制恢复。
5. **多套 Go 工具链互相干扰**：若系统中同时存在多个 `go.exe`，请将项目期望使用的 Go 安装目录（例如 `D:\go-legacy-win7\bin`）置于 `PATH` 最前面，避免旧版 Go 抢先解析。

### Q: 构建脚本提示 `gcc` 找不到 / MinGW 工具链未配置？
A: `build-all.bat` 与 `build-dll.bat` 默认依赖项目根目录下的 `mingw32` 与 `mingw64` 目录。请：

1. 下载 MinGW-w64，解压后将 32 位工具链重命名为 `mingw32`、64 位工具链重命名为 `mingw64`。
2. 确认两个目录的 `bin` 子目录下存在 `gcc.exe`、`g++.exe`、`ar.exe`、`ranlib.exe`。
3. 也可以改为使用系统全局安装的 MinGW，并将 `CC` / `CXX` / `AR` / `RANLIB` 环境变量指向相应路径。

### Q: 提示 `cannot find package "github.com/.../xxx"` 等网络相关错误？
A: 国内网络环境下，Go 默认的 `proxy.golang.org` 经常超时，可在执行构建前设置国内镜像：

```cmd
go env -w GOPROXY=https://goproxy.cn,direct
go env -w GOSUMDB=sum.golang.org
go mod download
```

### Q: DLL 调用崩溃或返回空指针？
A:

1. 加载 DLL 时请确保依赖的 `onnxruntime.dll`、`fastknNN.dll`、`fastonnx.dll` 与 `TextMiner.dll` 在**同一目录**。`build-dll.bat` 已自动完成拷贝；如果是手动部署，请保持这一布局。
2. DLL 导出函数 `TextMiner_ExtractFile` 返回的 JSON 字符串内存由 Go 通过 `C.CString`（底层 `malloc`）分配，调用方在使用完毕后必须用与 `malloc` 配对的方式释放：
   - C / C++：`free(result)`，不要使用 `delete`。
   - Python `ctypes`：`ctypes.free(result_ptr)`。
   - C#：`Marshal.FreeHGlobal`。
3. 32 位应用必须加载 32 位 DLL，64 位应用必须加载 64 位 DLL，位数不匹配会导致崩溃。

### Q: `cmd/TextMinerDLL/main.go` 中 `extractor.InitMagika(...)` 初始化被注释掉了，会不会有影响？
A: 不影响主体功能。被注释的代码段是早期在 DLL 启动时显式预热 Magika ONNX 会话的实验性逻辑。`extractor.ExtractFile` 在被调用时会按需懒加载模型，行为一致；如确有性能需要可手动取消注释。

### Q: 日志太多 / 占用磁盘过大？
A: 日志默认位于 `%APPDATA%\iandsec\logs`，按天轮转、最多保留 30 天。如需更激进清理，可在该目录下手动删除更早的 `.log` 文件，或修改 `pkg/logger/logger.go` 中的保留天数。

## 版本与变更

### 当前版本

`v1.0.0`（见 `cmd/TextMiner/main.go` 中的 `version` 变量）

### 变更日志

#### v1.0.0

- 首发版本，支持 90+ 种文件格式的内容提取
- 同时提供 32 位（x86）与 64 位（x64）命令行可执行文件
- 同时提供 32 位（x86）与 64 位（x64）Windows DLL，可供 C/C++/Python/C# 等语言调用
- 集成 Magika ONNX 文件类型检测
- 集成基于 ONNX Runtime 的 OCR 能力
- 支持 zip / 7z / rar / tar / gz / bz2 / xz / iso / rpm 等多种压缩格式
- Office 嵌入文件提取、加密检测
- Windows 7 SP1 兼容性（通过 `-DWINVER=0x0601` 等宏控制）

## 许可证

MIT

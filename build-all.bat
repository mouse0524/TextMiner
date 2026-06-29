@echo off
REM Unified Build Script - Windows 7 Compatible x86 and x64
REM This script builds both 32-bit and 64-bit versions with all dependencies

setlocal EnableDelayedExpansion

echo ========================================
echo Unified Build Script (x86 + x64)
echo ========================================

REM Create build directory structure
if exist build (
    echo Cleaning build directory...
    rmdir /S /Q build
)
mkdir build
mkdir build\x86
mkdir build\x64

echo Build directories created.
echo.

REM Build x86 version
echo ========================================
echo Building x86 (32-bit) Version
echo ========================================

set CGO_ENABLED=1
set GOOS=windows
set GOARCH=386

REM MinGW32 compiler setup
set MINGW32_PATH=%CD%\mingw32
set CC=%MINGW32_PATH%\bin\gcc.exe
set CXX=%MINGW32_PATH%\bin\g++.exe
set AR=%MINGW32_PATH%\bin\ar.exe
set RANLIB=%MINGW32_PATH%\bin\ranlib.exe
set PATH=%MINGW32_PATH%\bin;%PATH%

REM Set CGO compile flags - for Windows 7 SP1 compatibility
set CGO_CFLAGS=-I%CD%/models -DWINVER=0x0601 -D_WIN32_WINNT=0x0601 -D_WIN32_WINDOWS=0x0601 -DNTDDI_VERSION=0x06010000 -O2 -march=i686 -D_FILE_OFFSET_BITS=64
set CGO_LDFLAGS=-L%CD%/lib/x86 -lonnxruntime -static-libgcc -static-libstdc++ %CD%/build_helpers/dll_bootstrap.o
set LDFLAGS=-s -w

echo Building TextMiner-x86.exe...

REM Compile bootstrap C file
if exist %CD%/build_helpers/dll_bootstrap.o del /F /Q %CD%/build_helpers/dll_bootstrap.o >nul 2>&1
gcc -c %CD%/build_helpers/dll_bootstrap.c -o %CD%/build_helpers/dll_bootstrap.o -O2 -march=i686
if %ERRORLEVEL% NEQ 0 (
    echo Failed to compile bootstrap C file for x86
    goto :error
)

go build -tags "cgo onnxruntime" -ldflags="%LDFLAGS%" -o build\x86\TextMiner.exe ./cmd/TextMiner

if %ERRORLEVEL% EQU 0 (
    echo x86 build successful!
    
    REM Copy x86 dependencies to lib subdirectory
    echo Copying x86 dependencies...
    mkdir build\x86\lib 2>nul
    xcopy /Y /Q lib\x86\*.dll build\x86\lib\ >nul
    xcopy /Y /Q lib\x86\*.txt build\x86\lib\ >nul
    
    REM Copy models
    if not exist build\x86\models mkdir build\x86\models
    xcopy /Y /E /Q models build\x86\models\ >nul
    
    REM Copy config files
    xcopy /Y /Q *.json build\x86\ >nul 2>&1
    xcopy /Y /Q *.yaml build\x86\ >nul 2>&1
    xcopy /Y /Q *.yml build\x86\ >nul 2>&1
    
    echo x86 package complete!
) else (
    echo x86 build failed!
    goto :error
)

echo.
echo ========================================
echo Building x64 (64-bit) Version
echo ========================================

set CGO_ENABLED=1
set GOOS=windows
set GOARCH=amd64

REM MinGW64 compiler setup
set MINGW64_PATH=%CD%\mingw64
set CC=%MINGW64_PATH%\bin\gcc.exe
set CXX=%MINGW64_PATH%\bin\g++.exe
set AR=%MINGW64_PATH%\bin\ar.exe
set RANLIB=%MINGW64_PATH%\bin\ranlib.exe
set PATH=%MINGW64_PATH%\bin;%PATH%

REM Set CGO compile flags - for Windows 7 SP1 compatibility
set CGO_CFLAGS=-I%CD%/models -DWINVER=0x0601 -D_WIN32_WINNT=0x0601 -D_WIN32_WINDOWS=0x0601 -DNTDDI_VERSION=0x06010000 -O2 -march=x86-64
set CGO_LDFLAGS=-L%CD%/lib/x64 -lonnxruntime -static-libgcc -static-libstdc++ %CD%/build_helpers/dll_bootstrap.o
set LDFLAGS=-s -w

echo Building TextMiner-x64.exe...

REM Compile bootstrap C file
if exist %CD%/build_helpers/dll_bootstrap.o del /F /Q %CD%/build_helpers/dll_bootstrap.o >nul 2>&1
gcc -c %CD%/build_helpers/dll_bootstrap.c -o %CD%/build_helpers/dll_bootstrap.o -O2 -march=x86-64
if %ERRORLEVEL% NEQ 0 (
    echo Failed to compile bootstrap C file for x64
    goto :error
)

go build -tags "cgo onnxruntime" -ldflags="%LDFLAGS%" -o build\x64\TextMiner.exe ./cmd/TextMiner

if %ERRORLEVEL% EQU 0 (
    echo x64 build successful!

    REM Copy x64 dependencies to lib subdirectory
    echo Copying x64 dependencies...
    mkdir build\x64\lib 2>nul
    xcopy /Y /Q lib\x64\*.dll build\x64\lib\ >nul
    xcopy /Y /Q lib\x64\*.txt build\x64\lib\ >nul

    REM Copy required runtime DLLs next to TextMiner.exe
    REM (Windows loader only searches the executable directory for static imports,
    REM  the bootstrap SetDllDirectoryA runs too late to resolve them)
    echo Copying x64 runtime DLLs to executable directory...
    for %%F in (fastonnx.dll onnxruntime.dll onnxruntime_providers_shared.dll msvcp140.dll msvcp140_1.dll msvcp140_2.dll msvcp140_atomic_wait.dll msvcp140_codecvt_ids.dll vcruntime140.dll vcruntime140_1.dll vccorlib140.dll vcomp140.dll concrt140.dll ucrtbase.dll) do (
        if exist build\x64\lib\%%F copy /Y build\x64\lib\%%F build\x64\ >nul
    )

    REM Copy models
    if not exist build\x64\models mkdir build\x64\models
    xcopy /Y /E /Q models build\x64\models\ >nul

    REM Copy config files
    xcopy /Y /Q *.json build\x64\ >nul 2>&1
    xcopy /Y /Q *.yaml build\x64\ >nul 2>&1
    xcopy /Y /Q *.yml build\x64\ >nul 2>&1

    echo x64 package complete!
) else (
    echo x64 build failed!
    goto :error
)

echo.
echo ========================================
echo Build Summary
echo ========================================
echo x86 Package: build\x86\
echo x64 Package: build\x64\
xcopy /Y /Q build\x64\TextMiner.exe . >nul 2>&1
echo.
echo Both builds completed successfully!
echo.

echo ========================================
echo All Builds Complete!
echo ========================================

endlocal
exit /b 0

:error
echo.
echo ========================================
echo Build Failed!
echo ========================================
endlocal
exit /b 1

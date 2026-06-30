@echo off
setlocal enabledelayedexpansion

echo ========================================
echo Building DLL (x86 + x64)
echo ========================================

set BUILD_DIR=build-dll
set GOOS=windows

if exist "%BUILD_DIR%" (
    echo Cleaning build directory...
    rmdir /s /q "%BUILD_DIR%"
)

mkdir "%BUILD_DIR%\x86"
mkdir "%BUILD_DIR%\x64"

echo.
echo ========================================
echo Building x86 (32-bit) DLL
echo ========================================
set GOARCH=386
set CGO_ENABLED=1

set MINGW32_PATH=%CD%\mingw32
set CC=%MINGW32_PATH%\bin\gcc.exe
set CXX=%MINGW32_PATH%\bin\g++.exe
set AR=%MINGW32_PATH%\bin\ar.exe
set RANLIB=%MINGW32_PATH%\bin\ranlib.exe
set PATH=%MINGW32_PATH%\bin;%PATH%

set CGO_CFLAGS=-DWINVER=0x0601 -D_WIN32_WINNT=0x0601 -D_WIN32_WINDOWS=0x0601 -DNTDDI_VERSION=0x06010000 -O2 -march=i686 -D_FILE_OFFSET_BITS=64
set CGO_LDFLAGS=-static-libgcc -static-libstdc++ -L%CD%/lib/x86 -lonnxruntime
set LDFLAGS=-s -w

go build -buildmode=c-shared -o "%BUILD_DIR%\x86\TextMiner.dll" ./cmd/TextMinerdll

if %ERRORLEVEL% NEQ 0 (
    echo x86 DLL build failed
    exit /b 1
)

echo x86 DLL build successful

echo.
echo ========================================
echo Building x64 (64-bit) DLL
echo ========================================
set GOARCH=amd64
set CGO_ENABLED=1

set MINGW64_PATH=%CD%\mingw64
set CC=%MINGW64_PATH%\bin\gcc.exe
set CXX=%MINGW64_PATH%\bin\g++.exe
set AR=%MINGW64_PATH%\bin\ar.exe
set RANLIB=%MINGW64_PATH%\bin\ranlib.exe
set PATH=%MINGW64_PATH%\bin;%PATH%

set CGO_CFLAGS=-DWINVER=0x0601 -D_WIN32_WINNT=0x0601 -D_WIN32_WINDOWS=0x0601 -DNTDDI_VERSION=0x06010000 -O2 -D_FILE_OFFSET_BITS=64
set CGO_LDFLAGS=-static-libgcc -static-libstdc++ -L%CD%/lib/x64 -lonnxruntime
set LDFLAGS=-s -w

go build -buildmode=c-shared -o "%BUILD_DIR%\x64\TextMiner.dll" ./cmd/TextMinerdll

if %ERRORLEVEL% NEQ 0 (
    echo x64 DLL build failed
    exit /b 1
)

echo x64 DLL build successful

echo.
echo ========================================
echo Copying Dependencies
echo ========================================

echo Copying x86 dependencies...
xcopy /e /i /y lib\x86\* "%BUILD_DIR%\x86\" >nul 2>&1

echo Copying x64 dependencies...
xcopy /e /i /y lib\x64\* "%BUILD_DIR%\x64\" >nul 2>&1

echo Copying models to x86...
xcopy /e /i /y models "%BUILD_DIR%\x86\models\" >nul 2>&1

echo Copying models to x64...
xcopy /e /i /y models "%BUILD_DIR%\x64\models\" >nul 2>&1

echo Copying Python examples...
copy /y examples\python\TextMiner_example.py "%BUILD_DIR%\x86\TextMiner_example.py" >nul 2>&1
copy /y examples\python\TextMiner_example.py "%BUILD_DIR%\x64\TextMiner_example.py" >nul 2>&1
copy /y examples\python\test_TextMiner.py "%BUILD_DIR%\x86\test_TextMiner.py" >nul 2>&1
copy /y examples\python\test_TextMiner.py "%BUILD_DIR%\x64\test_TextMiner.py" >nul 2>&1

echo.
echo ========================================
echo Build Summary
echo ========================================
echo x86 DLL Package: %BUILD_DIR%\x86\
echo x64 DLL Package: %BUILD_DIR%\x64\

echo.
echo ========================================
echo All Builds Complete
echo ========================================

endlocal

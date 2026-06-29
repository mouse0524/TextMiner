@echo off
setlocal
set CGO_ENABLED=1
set CGO_CFLAGS=-I%CD%\models
set CGO_LDFLAGS=-L%CD%\models -lonnxruntime
go build -tags "cgo onnxruntime" -o TextMiner.exe cmd/TextMiner/main.go
if %ERRORLEVEL% EQU 0 (
    echo Build successful!
) else (
    echo Build failed!
    exit /b 1
)
endlocal

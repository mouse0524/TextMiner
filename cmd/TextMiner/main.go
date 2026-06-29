package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"textminer/pkg/extractor"
	"textminer/pkg/logger"

	"github.com/spf13/cobra"
)

var (
	version      = "v1.0.0"
	enableOcr    bool
	enableOutput bool
	rootCmd      = &cobra.Command{
		Use:   "dlp [文件路径]",
		Short: "文件内容提取工具，支持多种文件类型",
		Long: `文件内容提取工具，支持以下文件类型：
- 文本文件：txt（支持多种编码）
- Office文件：doc, docx, ppt, pptx, xls, xlsx
- PDF文件：pdf
- 代码文件：所有代码文件类型
- 图片文件：png, jpg, jpeg, bmp
- 压缩包：zip, 7z, rar, tar, gz, tgz, tar.gz
- 文本文件：txt等（支持多种编码）

功能说明：
- 文件类型检测使用Magika（基于ONNX Runtime）
- OCR识别：使用 --ocr 启用
		`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			filePath := args[0]
			absPath, _ := filepath.Abs(filePath)

			logger.Infof("开始提取文件: %s (OCR: %v)", absPath, enableOcr)

			if err := extractor.InitMagika(""); err != nil {
				logger.Warnf("初始化Magika失败: %v, 将使用默认检测器", err)
			}

			startTime := time.Now()

			// 提取文件内容
			result, err := extractor.ExtractFile(filePath, enableOcr)
			if err != nil {
				logger.Errorf("提取文件失败: %s, 错误: %v", absPath, err)
			}

			// 记录结束时间
			endTime := time.Now()
			duration := endTime.Sub(startTime)

			if result.Status == "success" {
				logger.Infof("提取成功: %s, 类型: %s, 耗时: %v, 内容长度: %d", absPath, result.FileType, duration, len(result.Content))
			} else {
				logger.Warnf("提取完成但有警告: %s, 状态: %s, 错误: %s", absPath, result.Status, result.ErrorMessage)
			}

			// 输出结果（JSON格式）
			jsonResult, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				logger.Errorf("JSON序列化失败: %v", err)
			}

			fmt.Println(string(jsonResult))

			// 将内容写入txt文件
			if enableOutput && result.Content != "" {
				outputFileName := result.FileName + ".txt"
				err := os.WriteFile(outputFileName, []byte(result.Content), 0644)
				if err != nil {
					logger.Errorf("写入txt文件失败: %s, 错误: %v", outputFileName, err)
				} else {
					logger.Infof("内容已写入文件: %s", outputFileName)
				}
			}
		},
	}

	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "打印工具版本信息",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("textminer 文件内容提取工具 %s\n", version)
		},
	}
)

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.Flags().BoolVar(&enableOcr, "ocr", false, "启用OCR识别（默认关闭）")
	rootCmd.Flags().BoolVar(&enableOutput, "output", false, "将提取内容写入txt文件（默认关闭）")

	logDir := filepath.Join(os.Getenv("APPDATA"), "iandsec", "logs")
	if err := logger.InitLogger(logDir); err != nil {
		fmt.Printf("初始化日志失败: %v\n", err)
	}
}

func main() {
	// 注册信号钩子：SIGINT/SIGTERM 时释放 OCR 引擎，避免 ONNX 句柄泄漏
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logger.Warnf("收到信号 %v, 正在关闭 OCR 引擎...", sig)
		_ = extractor.CloseOcrProcessor()
		os.Exit(130)
	}()
	defer func() {
		_ = extractor.CloseOcrProcessor()
	}()

	if err := rootCmd.Execute(); err != nil {
		_ = extractor.CloseOcrProcessor()
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

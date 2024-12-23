package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/562589540/bcut-asr-go/pkg/asr"
	"github.com/562589540/bcut-asr-go/pkg/types"
	"github.com/schollz/progressbar/v3"
)

var (
	inputFile  string
	outputFile string
	format     string
	interval   float64
)

func init() {
	flag.StringVar(&inputFile, "i", "", "输入文件路径")
	flag.StringVar(&outputFile, "o", "", "输出文件路径")
	flag.StringVar(&format, "f", "srt", "输出格式(srt/lrc/txt/json)")
	flag.Float64Var(&interval, "t", 5.0, "字幕断句时间间隔(秒)")
}

func main() {
	flag.Parse()

	if inputFile == "" {
		fmt.Println("请指定输入文件路径")
		flag.Usage()
		os.Exit(1)
	}

	// 创建进度条
	var (
		bar       *progressbar.ProgressBar
		lastStage types.ProgressStage
	)

	// 创建新进度条的函数
	newBar := func(msg string) *progressbar.ProgressBar {
		return progressbar.NewOptions64(100,
			progressbar.OptionFullWidth(),
			progressbar.OptionSetDescription(msg),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "=",
				SaucerHead:    ">",
				SaucerPadding: "-",
				BarStart:      "[",
				BarEnd:        "]",
			}),
		)
	}

	// 创建进度回调
	progress := func(info types.ProgressInfo) {
		// 生成新的消息
		prefix := ""
		switch info.Stage {
		case types.StageInit:
			prefix = "初始化"
		case types.StageUpload:
			prefix = "上传中"
		case types.StageProcess:
			prefix = "识别中"
		case types.StageComplete:
			prefix = "完成"
		}
		msg := fmt.Sprintf("[%s] %s", prefix, info.Description)

		// 只在阶段变化时创建新进度条
		if info.Stage != lastStage || bar == nil {
			if bar != nil {
				// 确保前一个进度条完成
				if lastStage != types.StageComplete {
					_ = bar.Set(100)
				}
				fmt.Println() // 换行，保留旧进度条
			}
			bar = newBar(msg)
			lastStage = info.Stage
		} else {
			// 更新消息
			bar.Describe(msg)
		}

		// 更新进度
		_ = bar.Set(info.Current)
	}

	// 设置转换选项
	options := asr.ConvertOptions{
		Format:     strings.ToLower(format),
		Interval:   interval,
		Progress:   progress,
		OutputPath: outputFile,
	}

	// 执行转换
	if err := asr.ConvertToSubtitle(inputFile, options); err != nil {
		fmt.Printf("\n转换失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n转换完成！输出文件: %s\n", getOutputPath(inputFile, outputFile, format))
}

func getOutputPath(inputFile, outputFile, format string) string {
	if outputFile != "" {
		if info, err := os.Stat(outputFile); err == nil && info.IsDir() {
			return filepath.Join(outputFile,
				filepath.Base(inputFile[:len(inputFile)-len(filepath.Ext(inputFile))])+"."+format)
		}
		return outputFile
	}
	return filepath.Join(
		filepath.Dir(inputFile),
		filepath.Base(inputFile[:len(inputFile)-len(filepath.Ext(inputFile))])+"."+format,
	)
}

package asr

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/562589540/bcut-asr-go/pkg/types"
)

func TestIntegration_FullProcess(t *testing.T) {

	// 跳过集成测试，除非明确指定要运行
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 从环境变量获取测试文件路径
	testFile := os.Getenv("TEST_FILE")
	if testFile == "" {
		t.Skip("未指定测试文件，请设置 TEST_FILE 环境变量")
	}

	// 创建ASR实例并设置进度回调
	bcutASR := New().WithProgress(func(info types.ProgressInfo) {
		switch info.Stage {
		case types.StageInit:
			t.Logf("[初始化] 进度: %d%%, %s", info.Current, info.Description)
		case types.StageUpload:
			t.Logf("[上传] 进度: %d%%, %s", info.Current, info.Description)
		case types.StageProcess:
			t.Logf("[识别] 进度: %d%%, %s", info.Current, info.Description)
		case types.StageComplete:
			t.Logf("[完成] 进度: %d%%, %s", info.Current, info.Description)
		}
	})

	// 设置输入文件
	t.Log("开始加载文件...")
	if err := bcutASR.SetData(testFile); err != nil {
		t.Fatalf("设置输入文件失败: %v", err)
	}
	t.Logf("文件加载成功，格式: %s", bcutASR.soundFormat)

	// 上传文件
	t.Log("开始上传文件...")
	if err := bcutASR.Upload(); err != nil {
		t.Fatalf("上传失败: %v", err)
	}
	t.Log("文件上传成功")

	// 创建任务
	t.Log("创建识别任务...")
	taskID, err := bcutASR.CreateTask()
	if err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	t.Logf("任务创建成功，ID: %s", taskID)

	// 轮询检查任务状态
	t.Log("等待识别结果...")
	for {
		result, err := bcutASR.QueryResult()
		if err != nil {
			t.Fatalf("查询结果失败: %v", err)
		}

		if result == nil {
			t.Log("识别中...")
			time.Sleep(5 * time.Second) // 避免请求过于频繁
			continue
		}

		// 测试不同格式的输出
		t.Run("SRT输出", func(t *testing.T) {
			srtOutput := result.ToSRT()
			outputFile := filepath.Join(
				filepath.Dir(testFile),
				filepath.Base(testFile[:len(testFile)-len(filepath.Ext(testFile))])+".srt",
			)
			if err := os.WriteFile(outputFile, []byte(srtOutput), 0644); err != nil {
				t.Errorf("写入SRT文件失败: %v", err)
			}
			t.Logf("SRT文件已保存: %s", outputFile)
		})

		t.Run("LRC输出", func(t *testing.T) {
			lrcOutput := result.ToLRC()
			outputFile := filepath.Join(
				filepath.Dir(testFile),
				filepath.Base(testFile[:len(testFile)-len(filepath.Ext(testFile))])+".lrc",
			)
			if err := os.WriteFile(outputFile, []byte(lrcOutput), 0644); err != nil {
				t.Errorf("写入LRC文件失败: %v", err)
			}
			t.Logf("LRC文件已保存: %s", outputFile)
		})

		t.Run("TXT输出", func(t *testing.T) {
			txtOutput := result.ToTXT()
			outputFile := filepath.Join(
				filepath.Dir(testFile),
				filepath.Base(testFile[:len(testFile)-len(filepath.Ext(testFile))])+".txt",
			)
			if err := os.WriteFile(outputFile, []byte(txtOutput), 0644); err != nil {
				t.Errorf("写入TXT文件失败: %v", err)
			}
			t.Logf("TXT文件已保存: %s", outputFile)
		})

		break
	}
}

func TestConvertToSubtitle(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	testFile := os.Getenv("TEST_FILE")
	if testFile == "" {
		t.Skip("未指定测试文件，请设置 TEST_FILE 1环境变量")
	}

	// 创建临时目录用于测试
	tempDir, err := os.MkdirTemp("", "bcut-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name    string
		options ConvertOptions
	}{
		// {
		// 	name:    "默认选项",
		// 	options: ConvertOptions{},
		// },
		{
			name: "指定输出目录",
			options: ConvertOptions{
				Format:     "srt",
				Interval:   5.0,
				OutputPath: tempDir,
				Progress: func(info types.ProgressInfo) {
					t.Logf("进度: %s %d%%, %s", info.Stage, info.Current, info.Description)
				},
			},
		},
		{
			name: "指定完整输出路径",
			options: ConvertOptions{
				Format:     "lrc",
				Interval:   5.0,
				OutputPath: filepath.Join(tempDir, "output.lrc"),
				Progress: func(info types.ProgressInfo) {
					t.Logf("进度: %s %d%%, %s", info.Stage, info.Current, info.Description)
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ConvertToSubtitle(testFile, tt.options); err != nil {
				t.Fatal(err)
			}
		})
	}
}

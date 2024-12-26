package asr

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/562589540/bcut-asr-go/pkg/types"
	"github.com/562589540/bcut-asr-go/pkg/utils"
)

type BcutASR struct {
	client      *http.Client
	soundName   string
	soundData   []byte
	soundFormat string
	inBossKey   string
	resourceID  string
	uploadID    string
	uploadURLs  []string
	perSize     int
	etags       []string
	downloadURL string
	taskID      string
	onProgress  types.ProgressCallback
	ctx         context.Context
}

func New(ctx context.Context, cookie ...string) *BcutASR {
	if ctx == nil {
		ctx = context.Background()
	}
	b := &BcutASR{
		client: &http.Client{},
		etags:  make([]string, 0),
		ctx:    ctx,
	}
	return b
}

func (b *BcutASR) processMedia(filePath string) error {
	ext := strings.ToLower(filepath.Ext(filePath))

	// 检查是否是支持的音频格式
	for _, format := range types.SupportedInputFormats {
		if "."+format == ext {
			// 直接读取音��文件
			if b.onProgress != nil {
				b.reportProgress(types.StageInit, 0, "读取音频文件...")
			}
			data, err := os.ReadFile(filePath)
			if err != nil {
				return err
			}
			b.soundData = data
			b.soundName = filepath.Base(filePath)
			b.soundFormat = format
			return nil
		}
	}

	// 不是支持的音频格式，尝试用ffmpeg提取音频
	b.reportProgress(types.StageInit, 20, "准备提取音频...")

	// 准备命令
	cmd := utils.RunCommand("ffmpeg",
		"-v", "warning",
		"-i", filePath,
		"-ac", "1",
		"-acodec", "aac",
		"-ar", "16000",
		//"-ab", "32k",
		"-f", "adts",
		"-",
	)

	// 创建缓冲区
	var buf bytes.Buffer
	cmd.Stdout = &buf

	// 创建stderr管道用于进度监控
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("创建stderr管道失败: %w", err)
	}

	// 启动进度监控
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "time=") {
				b.reportProgress(types.StageInit, 50, "音频提取中")
			}
		}
	}()

	// 执行命令
	b.reportProgress(types.StageInit, 40, "开始提取音频...")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg执行失败: %w", err)
	}

	// 保存结果
	b.soundData = buf.Bytes()
	if len(b.soundData) == 0 {
		return fmt.Errorf("未提取到音频数据")
	}

	b.soundName = strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath)) + ".aac"
	b.soundFormat = "aac"

	b.reportProgress(types.StageInit, 100, "音频提取完成")
	return nil
}

func (b *BcutASR) SetData(filePath string) error {
	if b.onProgress != nil {
		b.reportProgress(types.StageInit, 0, "开始加载文件...")
	}
	err := b.processMedia(filePath)
	if err == nil {
		return nil
	}
	return err
}

func (b *BcutASR) Upload() error {
	// 1. 申请上传
	formData := url.Values{
		"type":               {"2"},
		"name":               {b.soundName},
		"size":               {strconv.Itoa(len(b.soundData))},
		"resource_file_type": {b.soundFormat},
		"model_id":           {"7"},
	}

	req, err := http.NewRequestWithContext(b.ctx, http.MethodPost, types.GetAPIBaseURL()+types.APIReqUpload, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("request upload failed: %w", err)
	}
	defer resp.Body.Close()

	var result types.ASRResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode response failed: %w", err)
	}

	if result.Code != 0 {
		return fmt.Errorf("API error: %d - %s", result.Code, result.Message)
	}

	var createResp types.ResourceCreateResponse
	createRespData, err := json.Marshal(result.Data)
	if err := json.Unmarshal(createRespData, &createResp); err != nil {
		return fmt.Errorf("parse create response failed: %w", err)
	}

	b.inBossKey = createResp.InBossKey
	b.resourceID = createResp.ResourceID
	b.uploadID = createResp.UploadID
	b.uploadURLs = createResp.UploadURLs
	b.perSize = createResp.PerSize

	// 2. 分片上传
	if err := b.uploadParts(); err != nil {
		return fmt.Errorf("upload parts failed: %w", err)
	}

	// 3. 提交上传
	if err := b.commitUpload(); err != nil {
		return fmt.Errorf("commit upload failed: %w", err)
	}

	return nil
}

func (b *BcutASR) uploadParts() error {
	totalParts := len(b.uploadURLs)
	for i, url := range b.uploadURLs {
		select {
		case <-b.ctx.Done():
			return b.ctx.Err()
		default:
		}

		// 计算上传进度 (0-90%)
		progress := (i + 1) * 90 / totalParts
		b.reportProgress(
			types.StageUpload,
			progress,
			fmt.Sprintf("正在上传分片 %d/%d", i+1, totalParts),
		)

		start := i * b.perSize
		end := start + b.perSize
		if end > len(b.soundData) {
			end = len(b.soundData)
		}

		req, err := http.NewRequestWithContext(b.ctx, http.MethodPut, url, bytes.NewReader(b.soundData[start:end]))
		if err != nil {
			return err
		}

		resp, err := b.client.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()

		etag := resp.Header.Get("Etag")
		if etag == "" {
			return fmt.Errorf("no etag in response for part %d", i)
		}
		b.etags = append(b.etags, etag)
	}

	// 最后一次性报告100%
	b.reportProgress(types.StageUpload, 100, "上传完成")
	return nil
}

func (b *BcutASR) commitUpload() error {
	formData := url.Values{
		"in_boss_key": {b.inBossKey},
		"resource_id": {b.resourceID},
		"etags":       {strings.Join(b.etags, ",")},
		"upload_id":   {b.uploadID},
		"model_id":    {"7"},
	}

	req, err := http.NewRequestWithContext(b.ctx, http.MethodPost, types.GetAPIBaseURL()+types.APICommitUpload, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result types.ASRResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if result.Code != 0 {
		return fmt.Errorf("API error: %d - %s", result.Code, result.Message)
	}

	var completeResp types.ResourceCompleteResponse
	completeRespData, err := json.Marshal(result.Data)
	if err := json.Unmarshal(completeRespData, &completeResp); err != nil {
		return err
	}

	b.downloadURL = completeResp.DownloadURL
	return nil
}

func (b *BcutASR) CreateTask() (string, error) {
	reqData := map[string]interface{}{
		"resource": b.downloadURL,
		"model_id": "7",
	}

	reqBody, err := json.Marshal(reqData)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(b.ctx, http.MethodPost,
		types.GetAPIBaseURL()+types.APICreateTask,
		bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := b.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result types.ASRResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Code != 0 {
		return "", fmt.Errorf("API error: %d - %s", result.Code, result.Message)
	}

	var taskResp types.TaskCreateResponse
	taskRespData, err := json.Marshal(result.Data)
	if err := json.Unmarshal(taskRespData, &taskResp); err != nil {
		return "", err
	}

	b.taskID = taskResp.TaskID
	return taskResp.TaskID, nil
}

func (b *BcutASR) QueryResult() (*types.ASRResult, error) {
	req, err := http.NewRequestWithContext(b.ctx, http.MethodGet,
		fmt.Sprintf("%s%s?model_id=7&task_id=%s",
			types.GetAPIBaseURL(), types.APIQueryResult, b.taskID),
		nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result types.ASRResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("API error: %d - %s", result.Code, result.Message)
	}

	var taskResult types.TaskResultResponse
	taskResultData, err := json.Marshal(result.Data)
	if err := json.Unmarshal(taskResultData, &taskResult); err != nil {
		return nil, err
	}

	switch taskResult.State {
	case types.StateStop: // 0 - 排队中
		b.reportProgress(types.StageProcess, 50, "排队中...")
	case types.StateRunning: // 1 - 处理中
		b.reportProgress(types.StageProcess, 75, "正在识别...")
	case types.StateError: // 3 - 失败
		return nil, fmt.Errorf("task failed: %s", taskResult.Remark)
	case types.StateComplete: // 4 - 完成
		b.reportProgress(types.StageComplete, 100, "识别完成")
	}

	if taskResult.State != types.StateComplete {
		return nil, nil
	}

	var asrResult types.ASRResult
	if err := json.Unmarshal([]byte(taskResult.Result), &asrResult); err != nil {
		return nil, err
	}

	return &asrResult, nil
}

func (b *BcutASR) WithProgress(callback types.ProgressCallback) *BcutASR {
	b.onProgress = callback
	return b
}

func (b *BcutASR) reportProgress(stage types.ProgressStage, current int, description string) {
	if b.onProgress != nil {
		b.onProgress(types.ProgressInfo{
			Stage:       stage,
			Total:       100,
			Current:     current,
			Description: description,
		})
	}
}

// ConvertOptions 转换选项
type ConvertOptions struct {
	Format     string                 // 输出格式，默认 "srt"
	Interval   float64                // 轮询间隔（秒），默认 30.0
	Progress   types.ProgressCallback // 进度回调，可选
	OutputPath string                 // 输出路径，可选，默认与输入文件同目录
	Context    context.Context        // 上下文，可选，用于取消操作
}

// DefaultConvertOptions 默认转换选项
var DefaultConvertOptions = ConvertOptions{
	Format:   "srt",
	Interval: 30.0,
}

// ConvertToSubtitle 快捷转换方法
func ConvertToSubtitle(inputFile string, opts ...ConvertOptions) error {
	// 使用默认选项
	options := DefaultConvertOptions
	// 如果提供了选项，则使用提供的选项
	if len(opts) > 0 {
		options = opts[0]
	}
	// 确保格式有值
	if options.Format == "" {
		options.Format = "srt"
	}
	// 确保间隔有值
	if options.Interval <= 0 {
		options.Interval = 30.0
	}
	// 确保上下文有值
	if options.Context == nil {
		options.Context = context.Background()
	}

	bcutASR := New(options.Context).WithProgress(options.Progress)

	// 设置输入文件
	if err := bcutASR.SetData(inputFile); err != nil {
		return err
	}

	// 上传文件
	if err := bcutASR.Upload(); err != nil {
		return err
	}

	// 创建任务
	taskID, err := bcutASR.CreateTask()
	if err != nil {
		return err
	}
	if options.Progress != nil {
		options.Progress(types.ProgressInfo{
			Stage:       types.StageProcess,
			Total:       100,
			Current:     25,
			Description: fmt.Sprintf("任务已创建: %s", taskID),
		})
	}

	// 轮询检查任务状态
	ticker := time.NewTicker(time.Duration(options.Interval * float64(time.Second)))
	defer ticker.Stop()

	for {
		select {
		case <-options.Context.Done():
			return options.Context.Err()
		case <-ticker.C:
			result, err := bcutASR.QueryResult()
			if err != nil {
				return err
			}

			if result == nil {
				continue
			}

			// 生成输出文件名
			var outputFile string
			if options.OutputPath != "" {
				// 如果指定了输出路径
				if err := os.MkdirAll(filepath.Dir(options.OutputPath), 0755); err != nil {
					return fmt.Errorf("创建输出目录失败: %w", err)
				}
				// 如果指定的是目录，则在该目录下生成默认文件名
				if info, err := os.Stat(options.OutputPath); err == nil && info.IsDir() {
					outputFile = filepath.Join(options.OutputPath,
						filepath.Base(inputFile[:len(inputFile)-len(filepath.Ext(inputFile))])+"."+options.Format)
				} else {
					// 否则使用指定的完整路径
					outputFile = options.OutputPath
				}
			} else {
				// 默认与输入文件同目录
				outputFile = filepath.Join(
					filepath.Dir(inputFile),
					filepath.Base(inputFile[:len(inputFile)-len(filepath.Ext(inputFile))])+"."+options.Format,
				)
			}

			// 根据格式输出结果
			var output string
			switch options.Format {
			case "srt":
				output = result.ToSRT()
			case "lrc":
				output = result.ToLRC()
			case "txt":
				output = result.ToTXT()
			case "json":
				jsonBytes, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return fmt.Errorf("JSON序列化失败: %w", err)
				}
				output = string(jsonBytes)
			default:
				return fmt.Errorf("不支持的输出格式: %s", options.Format)
			}

			return os.WriteFile(outputFile, []byte(output), 0644)
		}
	}
}

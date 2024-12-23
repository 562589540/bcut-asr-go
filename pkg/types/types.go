package types

import (
	"fmt"
	"sync"
)

var (
	apiBaseURLMu sync.RWMutex
	apiBaseURL   = "https://member.bilibili.com/x/bcut/rubick-interface"
)

// GetAPIBaseURL 获取API基础URL
func GetAPIBaseURL() string {
	apiBaseURLMu.RLock()
	defer apiBaseURLMu.RUnlock()
	return apiBaseURL
}

// SetAPIBaseURL 设置API基础URL
func SetAPIBaseURL(url string) {
	apiBaseURLMu.Lock()
	apiBaseURL = url
	apiBaseURLMu.Unlock()
}

const (
	APIReqUpload    = "/resource/create"
	APICommitUpload = "/resource/create/complete"
	APICreateTask   = "/task"
	APIQueryResult  = "/task/result"
)

var (
	SupportedInputFormats  = []string{"flac", "aac", "m4a", "mp3", "wav"}
	SupportedOutputFormats = []string{"srt", "json", "lrc", "txt"}
)

type ResultState int

const (
	StateStop     ResultState = 0
	StateRunning  ResultState = 1
	StateError    ResultState = 3
	StateComplete ResultState = 4
)

// ASRResponse 定义API响应结构
type ASRResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// ResourceCreateResponse 定义上传请响应
type ResourceCreateResponse struct {
	ResourceID string   `json:"resource_id"`
	Title      string   `json:"title"`
	Type       int      `json:"type"`
	InBossKey  string   `json:"in_boss_key"`
	Size       int64    `json:"size"`
	UploadURLs []string `json:"upload_urls"`
	UploadID   string   `json:"upload_id"`
	PerSize    int      `json:"per_size"`
}

// ASRResult 定义识别结果
type ASRResult struct {
	Utterances []Utterance `json:"utterances"`
	Version    string      `json:"version"`
}

// Utterance 定义识别的句子
type Utterance struct {
	StartTime  int64   `json:"start_time"`
	EndTime    int64   `json:"end_time"`
	Transcript string  `json:"transcript"`
	Words      []Words `json:"words"`
}

// Words 定义识别的词
type Words struct {
	Label     string `json:"label"`
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
}

// ResourceCompleteResponse 定义上传完成响应
type ResourceCompleteResponse struct {
	ResourceID  string `json:"resource_id"`
	DownloadURL string `json:"download_url"`
}

// TaskCreateResponse 定义任务创建响应
type TaskCreateResponse struct {
	Resource string `json:"resource"`
	Result   string `json:"result"`
	TaskID   string `json:"task_id"`
}

// TaskResultResponse 义任务结果响应
type TaskResultResponse struct {
	TaskID string      `json:"task_id"`
	Result string      `json:"result"`
	Remark string      `json:"remark"`
	State  ResultState `json:"state"`
}

// ToSRT 将识别结果转换为SRT格式
func (r *ASRResult) ToSRT() string {
	var result string
	for i, u := range r.Utterances {
		result += fmt.Sprintf("%d\n%s\n%s\n\n",
			i+1,
			formatSRTTimestamp(u.StartTime, u.EndTime),
			u.Transcript)
	}
	return result
}

// ToLRC 将识别结果转换为LRC格式
func (r *ASRResult) ToLRC() string {
	var result string
	for _, u := range r.Utterances {
		result += fmt.Sprintf("[%s]%s\n",
			formatLRCTimestamp(u.StartTime),
			u.Transcript)
	}
	return result
}

// ToTXT 将识别结果转换为纯文本格式
func (r *ASRResult) ToTXT() string {
	var result string
	for _, u := range r.Utterances {
		result += u.Transcript + "\n"
	}
	return result
}

func formatSRTTimestamp(start, end int64) string {
	return fmt.Sprintf("%02d:%02d:%02d,%03d --> %02d:%02d:%02d,%03d",
		start/3600000, (start/60000)%60, (start/1000)%60, start%1000,
		end/3600000, (end/60000)%60, (end/1000)%60, end%1000)
}

func formatLRCTimestamp(ts int64) string {
	return fmt.Sprintf("%02d:%02d.%02d",
		ts/60000, (ts/1000)%60, (ts%1000)/10)
}

// ProgressStage 进度阶段
type ProgressStage string

const (
	StageInit     ProgressStage = "init"     // 初始化
	StageUpload   ProgressStage = "upload"   // 上传文件
	StageProcess  ProgressStage = "process"  // 语音识别
	StageComplete ProgressStage = "complete" // 完成
)

// ProgressInfo 进度信息
type ProgressInfo struct {
	Stage       ProgressStage // 当前阶段
	Total       int           // 总进度（100）
	Current     int           // 当前进度（0-100）
	Description string        // 当前状态描述
}

// ProgressCallback 进度回调函数类型
type ProgressCallback func(ProgressInfo)

// 进度中文
func ProgressStageCN(stage ProgressStage) string {
	prefix := ""
	switch stage {
	case StageInit:
		prefix = "初始化"
	case StageUpload:
		prefix = "上传中"
	case StageProcess:
		prefix = "识别中"
	case StageComplete:
		prefix = "完成"
	}
	return prefix
}

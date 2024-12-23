package asr

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/562589540/bcut-asr-go/pkg/types"
)

func TestBcutASR_SetData(t *testing.T) {
	// 创建临时测试文件
	tmpFile := filepath.Join(t.TempDir(), "test.mp3")
	if err := os.WriteFile(tmpFile, []byte("test data"), 0644); err != nil {
		t.Fatal(err)
	}

	asr := New()
	if err := asr.SetData(tmpFile); err != nil {
		t.Errorf("SetData() error = %v", err)
	}

	if asr.soundName != "test.mp3" {
		t.Errorf("soundName = %v, want %v", asr.soundName, "test.mp3")
	}

	if asr.soundFormat != "mp3" {
		t.Errorf("soundFormat = %v, want %v", asr.soundFormat, "mp3")
	}
}

func TestBcutASR_Upload(t *testing.T) {
	// 模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/resource/create":
			// 模拟申请上传响应
			resp := types.ASRResponse{
				Code: 0,
				Data: types.ResourceCreateResponse{
					ResourceID: "test-resource",
					UploadURLs: []string{"http://test.com/upload"},
					UploadID:   "test-upload",
					InBossKey:  "test-key",
					PerSize:    1024,
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "/resource/create/complete":
			// 模拟提交上传响应
			resp := types.ASRResponse{
				Code: 0,
				Data: types.ResourceCompleteResponse{
					ResourceID:  "test-resource",
					DownloadURL: "http://test.com/download",
				},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	// 修改API基础URL为测试服务器
	origBaseURL := types.GetAPIBaseURL()
	types.SetAPIBaseURL(server.URL)
	defer func() { types.SetAPIBaseURL(origBaseURL) }()

	asr := New()
	asr.soundName = "test.mp3"
	asr.soundData = []byte("test data")
	asr.soundFormat = "mp3"

	if err := asr.Upload(); err != nil {
		t.Errorf("Upload() error = %v", err)
	}

	if asr.downloadURL != "http://test.com/download" {
		t.Errorf("downloadURL = %v, want %v", asr.downloadURL, "http://test.com/download")
	}
}

func TestBcutASR_CreateTask(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := types.ASRResponse{
			Code: 0,
			Data: types.TaskCreateResponse{
				TaskID: "test-task",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	origBaseURL := types.GetAPIBaseURL()
	types.SetAPIBaseURL(server.URL)
	defer func() { types.SetAPIBaseURL(origBaseURL) }()

	asr := New()
	asr.downloadURL = "http://test.com/download"

	taskID, err := asr.CreateTask()
	if err != nil {
		t.Errorf("CreateTask() error = %v", err)
	}

	if taskID != "test-task" {
		t.Errorf("taskID = %v, want %v", taskID, "test-task")
	}
}

func TestBcutASR_QueryResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := types.ASRResponse{
			Code: 0,
			Data: types.TaskResultResponse{
				TaskID: "test-task",
				State:  types.StateComplete,
				Result: `{
					"utterances": [
						{
							"start_time": 0,
							"end_time": 1000,
							"transcript": "test transcript",
							"words": []
						}
					],
					"version": "1.0"
				}`,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	origBaseURL := types.GetAPIBaseURL()
	types.SetAPIBaseURL(server.URL)
	defer func() { types.SetAPIBaseURL(origBaseURL) }()

	asr := New()
	asr.taskID = "test-task"

	result, err := asr.QueryResult()
	if err != nil {
		t.Errorf("QueryResult() error = %v", err)
	}

	if result == nil {
		t.Error("QueryResult() returned nil result")
		return
	}

	if len(result.Utterances) != 1 {
		t.Errorf("len(result.Utterances) = %v, want %v", len(result.Utterances), 1)
	}

	if result.Utterances[0].Transcript != "test transcript" {
		t.Errorf("transcript = %v, want %v", result.Utterances[0].Transcript, "test transcript")
	}
}

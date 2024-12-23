package types

import (
	"testing"
)

func TestASRResult_ToSRT(t *testing.T) {
	result := &ASRResult{
		Utterances: []Utterance{
			{
				StartTime:  1000,
				EndTime:    2000,
				Transcript: "测试字幕1",
			},
			{
				StartTime:  3000,
				EndTime:    4000,
				Transcript: "测试字幕2",
			},
		},
	}

	expected := "1\n00:00:01,000 --> 00:00:02,000\n测试字幕1\n\n" +
		"2\n00:00:03,000 --> 00:00:04,000\n测试字幕2\n\n"

	if got := result.ToSRT(); got != expected {
		t.Errorf("ToSRT() = %v, want %v", got, expected)
	}
}

func TestASRResult_ToLRC(t *testing.T) {
	result := &ASRResult{
		Utterances: []Utterance{
			{
				StartTime:  61000, // 1分1秒
				EndTime:    62000,
				Transcript: "测试字幕1",
			},
			{
				StartTime:  122000, // 2分2秒
				EndTime:    123000,
				Transcript: "测试字幕2",
			},
		},
	}

	expected := "[01:01.00]测试字幕1\n[02:02.00]测试字幕2\n"

	if got := result.ToLRC(); got != expected {
		t.Errorf("ToLRC() = %v, want %v", got, expected)
	}
}

func TestASRResult_ToTXT(t *testing.T) {
	result := &ASRResult{
		Utterances: []Utterance{
			{
				Transcript: "测试字幕1",
			},
			{
				Transcript: "测试字幕2",
			},
		},
	}

	expected := "测试字幕1\n测试字幕2\n"

	if got := result.ToTXT(); got != expected {
		t.Errorf("ToTXT() = %v, want %v", got, expected)
	}
}

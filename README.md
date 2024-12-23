# Bcut-ASR-Go

必剪语音识别的Go语言实现版本，支持云端语音字幕识别。

## 特性

- 支持直接上传 flac、aac、m4a、mp3、wav 音频格式
- 自动调用 ffmpeg 提取视频文件的音轨并转换为 aac 格式
- 支持 srt、json、lrc、txt 格式字幕输出
- 支持自定义断句时间间隔
- 支持标准输出

## 安装

确保已安装 FFmpeg，然后：

```bash
go install github.com/yourusername/bcut-asr-go/cmd/bcut-asr@latest
```

## 使用方法

### 命令行参数

```
-i  输入文件路径
-o  输出文件路径（可选，默认与输入文件同目录）
-f  输出格式，支持 srt/lrc/txt/json（可选，默认为srt）
-t  字幕断句时间间隔，单位秒（可选，默认为5.0）
```

### 命令行示例

```bash
# 基本用法
bcut-asr -i video.mp4

# 指定输出格式和文件
bcut-asr -i video.mp4 -f srt -o subtitle.srt

# 自定义断句时间间隔
bcut-asr -i video.mp4 -t 3.5

# 完整参数示例
bcut-asr -i video.mp4 -o output.srt -f srt -t 4.0
```

### 作为库使用

```go
package main

import (
    "log"
    
    "github.com/yourusername/bcut-asr-go/pkg/asr"
)

func main() {
    // 使用 ConvertOptions 进行转换
    options := asr.ConvertOptions{
        Format:     "srt",
        Interval:   5.0,        // 断句时间间隔（秒）
        OutputPath: "output.srt",
        Progress: func(info types.ProgressInfo) {
            // 处理进度回调
            log.Printf("进度: %d%%, %s", info.Current, info.Description)
        },
    }
    
    if err := asr.ConvertToSubtitle("input.mp4", options); err != nil {
        log.Fatal(err)
    }
}
```

## 进度回调

转换过程中会通过 Progress 回调函数报告进度，包含以下阶段：

- StageInit: 初始化阶段
- StageUpload: 文件上传阶段
- StageProcess: 语音识别阶段
- StageComplete: 完成阶段

每个阶段都会提供当前进度百分比和描述信息。

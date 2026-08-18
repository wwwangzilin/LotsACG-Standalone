package mediatool

import (
	"fmt"
	"image"
	"os"
	"strconv"

	"github.com/krau/ffmpeg-go"
)

// 使用 ffmpeg 压缩图片。quality>0 时通过 -q:v 控制 jpeg 画质 (数值越大画质越低)。
func compressImageByFFmpeg(inputPath, outputPath string, maxEdgeLength, quality int) error {
	file, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer file.Close()
	img, _, err := image.DecodeConfig(file)
	if err != nil {
		return err
	}
	outArgs := ffmpeg.KwArgs{}
	if quality > 0 {
		outArgs["q:v"] = strconv.Itoa(quality)
	}
	if maxEdgeLength > 0 {
		if img.Width > int(maxEdgeLength) || img.Height > int(maxEdgeLength) {
			if img.Width > img.Height {
				outArgs["vf"] = fmt.Sprintf("scale=%d:-1:flags=lanczos", maxEdgeLength)
			} else {
				outArgs["vf"] = fmt.Sprintf("scale=-1:%d:flags=lanczos", maxEdgeLength)
			}
		}
	}
	if err := ffmpeg.Input(inputPath).Output(outputPath, outArgs).OverWriteOutput().Run(); err != nil {
		return fmt.Errorf("failed to compress image: %w", err)
	}
	return nil
}

package mediatool

import (
	"fmt"
	"image"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/osutil"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/strutil"
)

var (
	ffmpegAvailable bool
	vipsFormat      map[string]struct{}
	nativeFormat    = map[string]struct{}{"jpeg": {}, "jpg": {}, "png": {}, "webp": {}, "avif": {}}
)

func init() {
	switch runtime.GOOS {
	case "windows":
		_, err := exec.LookPath("ffmpeg.exe")
		if err == nil {
			ffmpegAvailable = true
		}
	default:
		_, err := exec.LookPath("ffmpeg")
		if err == nil {
			ffmpegAvailable = true
		}
	}
}

func FFmpegAvailable() bool {
	return ffmpegAvailable
}

func GetImgSize(img image.Image) (int, int, error) {
	if img == nil {
		return 0, 0, fmt.Errorf("nil image")
	}
	bounds := img.Bounds()
	if bounds.Empty() {
		return 0, 0, fmt.Errorf("empty image")
	}
	return bounds.Dx(), bounds.Dy(), nil
}

func GetImgSizeFromReader(r io.Reader) (int, int, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to decode image: %w", err)
	}
	return GetImgSize(img)
}

// CompressImg compresses the image at inputPath and saves the result to outputPath.
//
// The input image will be resized so that its longest edge does not exceed maxEdgeLength,
//
// If the maxEdgeLength <= 0, no resizing will be performed.
func CompressImg(inputPath, outputPath, format string, maxEdgeLength int) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), os.ModePerm); err != nil {
		return err
	}
	if _, ok := vipsFormat[format]; ok {
		log.Debug("compressing image", "method", "vips", "input", inputPath, "output", outputPath, "format", format)
		err := compressImageVIPS(inputPath, outputPath, format, maxEdgeLength)
		if err != nil {
			return fmt.Errorf("failed to compress image with vips: %w", err)
		}
		return nil
	}
	if ffmpegAvailable {
		log.Debug("compressing image", "method", "ffmpeg", "input", inputPath, "output", outputPath, "format", format)
		err := compressImageByFFmpeg(inputPath, outputPath, maxEdgeLength, 0)
		if err != nil {
			return fmt.Errorf("failed to compress image with ffmpeg: %w", err)
		}
		return nil
	}
	if _, ok := nativeFormat[format]; ok {
		log.Debug("compressing image", "method", "native", "input", inputPath, "output", outputPath, "format", format)
		err := compressImageNative(inputPath, outputPath, format, maxEdgeLength, 0)
		if err != nil {
			return fmt.Errorf("failed to compress image with native: %w", err)
		}
		return nil
	}
	return fmt.Errorf("unsupported image format: %s", format)
}

func CompressImgForTelegram(input []byte) ([]byte, error) {
	if _, ok := vipsFormat["jpeg"]; ok {
		return compressImageForTelegramByVIPS(input)
	}
	tmpFile, err := os.CreateTemp(runtimecfg.Get().Storage.CacheDir, "mediatool_*.png")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	distFile, err := os.CreateTemp(runtimecfg.Get().Storage.CacheDir, "mediatool_*.jpg")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(distFile.Name())
	defer distFile.Close()

	err = os.WriteFile(tmpFile.Name(), input, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	if ffmpegAvailable {
		err = compressImageByFFmpeg(tmpFile.Name(), distFile.Name(), TelegramMaxPhotoSideLength, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to compress image by ffmpeg: %w", err)
		}
		result, err := os.ReadFile(distFile.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to read temp file: %w", err)
		}
		return result, nil
	}
	err = compressImageNative(tmpFile.Name(), distFile.Name(), "jpeg", TelegramMaxPhotoSideLength, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to compress image natively: %w", err)
	}
	result, err := os.ReadFile(distFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read temp file: %w", err)
	}
	return result, nil
}

func CompressImgForTelegramFromFile(filePath string) (*osutil.TempFile, error) {
	return CompressImgForTelegramFromFileLevel(filePath, 0)
}

// CompressImgForTelegramFromFileLevel 按指定力度压缩图片供 Telegram 上传。
// level 越大压缩越狠 (边长更小、画质更低), 用于 "file is too big" 时逐步降级重试。
func CompressImgForTelegramFromFileLevel(filePath string, level int) (*osutil.TempFile, error) {
	if level < 0 {
		level = 0
	}
	if level >= len(TelegramCompressLevels) {
		level = len(TelegramCompressLevels) - 1
	}
	lv := TelegramCompressLevels[level]
	outputPath := filepath.Join(runtimecfg.Get().Storage.CacheDir, "compress", fmt.Sprintf("tg_%s_%d.jpg", strutil.MD5Hash(filePath), rand.Int()))
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create compress directory: %w", err)
	}
	if _, ok := vipsFormat["jpeg"]; ok {
		err := compressImageForTelegramByVIPSFromFile(filePath, outputPath)
		if err != nil {
			return nil, err
		}
		f, err := os.Open(outputPath)
		if err != nil {
			return nil, err
		}
		return &osutil.TempFile{File: f}, nil
	}
	if ffmpegAvailable {
		err := compressImageByFFmpeg(filePath, outputPath, lv.MaxEdge, lv.FFmpegQ)
		if err != nil {
			return nil, err
		}
		f, err := os.Open(outputPath)
		if err != nil {
			return nil, err
		}
		return &osutil.TempFile{File: f}, nil
	}
	err := compressImageNative(filePath, outputPath, "jpeg", lv.MaxEdge, lv.NativeQ)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(outputPath)
	if err != nil {
		return nil, err
	}
	return &osutil.TempFile{File: f}, nil
}

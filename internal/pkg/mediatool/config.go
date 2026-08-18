package mediatool

const (
	TelegramMaxPhotoFileSize        int = 10 * 1024 * 1024
	TelegramMaxPhotoTotalSideLength int = 10000
	TelegramMaxPhotoSideLength      int = 2560
)

// TelegramCompressLevel 定义一次图片压缩的力度。
// MaxEdge: 最长边限制; FFmpegQ: ffmpeg -q:v 量化参数 (越大画质越低); NativeQ: 原生 jpeg 质量 (越小越低)。
type TelegramCompressLevel struct {
	MaxEdge int
	FFmpegQ int
	NativeQ int
}

// TelegramCompressLevels 逐级加大的压缩力度, 用于 Telegram "file is too big" 时重试。
var TelegramCompressLevels = []TelegramCompressLevel{
	{TelegramMaxPhotoSideLength, 0, 85}, // 默认
	{1600, 8, 70},
	{1024, 12, 60},
	{800, 16, 50},
}

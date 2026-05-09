package ocr

import "context"

// Client OCR客户端接口
type Client interface {
	// Recognize 通用文字识别（本地图片base64或URL）
	Recognize(ctx context.Context, images []string, mode OCRMode) (*OCRResult, error)

	// RecognizeWithURL 通过图片URL识别
	RecognizeWithURL(ctx context.Context, imageURLs []string, mode OCRMode) (*OCRResult, error)

	// RecognizeHandwriting 手写体识别
	RecognizeHandwriting(ctx context.Context, imageBase64List []string) (*OCRResult, error)

	// RecognizeTable 表格识别
	RecognizeTable(ctx context.Context, imageBase64 string) (*OCRResult, error)
}

// ImageQualityChecker 图片质量检测器
type ImageQualityChecker interface {
	// Filter 过滤不合格图片
	Filter(ctx context.Context, imageURLs []string) (valid []string, invalid []InvalidImage)
}

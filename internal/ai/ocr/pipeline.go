package ocr

import (
	"context"
	"fmt"

	"github.com/edaptix/server/internal/config"
	"go.uber.org/zap"
)

// Pipeline OCR处理流水线（本地PaddleOCR-VL优先 + 百度OCR云端降级）
type Pipeline struct {
	cloudOCR  *BaiduOCRClient
	imageQA   *SimpleImageQualityChecker
	engine    string // 当前使用的引擎
}

// NewPipeline 创建OCR流水线
// 当前阶段：仅百度OCR API（PaddleOCR-VL本地部署后接入）
func NewPipeline(cfg config.OCRConfig) *Pipeline {
	return &Pipeline{
		cloudOCR: NewBaiduOCRClient(cfg),
		imageQA:  NewSimpleImageQualityChecker(),
		engine:   cfg.Engine,
	}
}

// ProcessOCR OCR处理流水线主入口
func (p *Pipeline) ProcessOCR(ctx context.Context, images []string, mode OCRMode) (*OCRResult, error) {
	// Step 1: 图片质量预检
	validImages, invalidImages := p.imageQA.Filter(ctx, images)

	if len(validImages) == 0 {
		return nil, fmt.Errorf("all images are invalid")
	}

	// Step 2: 调用OCR引擎识别
	var result *OCRResult
	var err error

	switch mode {
	case OCRModeCatalog:
		result, err = p.recognizeCatalog(ctx, validImages)
	case OCRModeHomework:
		result, err = p.recognizeHomework(ctx, validImages)
	case OCRModeHandwriting:
		result, err = p.recognizeHandwriting(ctx, validImages)
	default:
		result, err = p.recognizeCatalog(ctx, validImages)
	}

	if err != nil {
		zap.L().Error("OCR pipeline failed",
			zap.String("mode", string(mode)),
			zap.Error(err),
		)
		return nil, err
	}

	// Step 3: 合并无效图片信息
	result.InvalidImages = invalidImages
	return result, nil
}

// recognizeCatalog 教材目录识别
func (p *Pipeline) recognizeCatalog(ctx context.Context, images []string) (*OCRResult, error) {
	result, err := p.cloudOCR.Recognize(ctx, images, OCRModeCatalog)
	if err != nil {
		return nil, fmt.Errorf("catalog OCR failed: %w", err)
	}
	return result, nil
}

// recognizeHomework 作业识别
func (p *Pipeline) recognizeHomework(ctx context.Context, images []string) (*OCRResult, error) {
	result, err := p.cloudOCR.Recognize(ctx, images, OCRModeHomework)
	if err != nil {
		return nil, fmt.Errorf("homework OCR failed: %w", err)
	}
	return result, nil
}

// recognizeHandwriting 手写体识别
func (p *Pipeline) recognizeHandwriting(ctx context.Context, images []string) (*OCRResult, error) {
	result, err := p.cloudOCR.RecognizeHandwriting(ctx, images)
	if err != nil {
		return nil, fmt.Errorf("handwriting OCR failed: %w", err)
	}
	return result, nil
}

// GetClient 获取底层OCR客户端（供Service层直接调用）
func (p *Pipeline) GetClient() *BaiduOCRClient {
	return p.cloudOCR
}

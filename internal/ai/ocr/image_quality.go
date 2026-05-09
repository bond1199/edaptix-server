package ocr

import (
	"context"
	"path/filepath"
	"strings"
)

// SimpleImageQualityChecker 简单图片质量检测器
// 当前阶段：基础格式和命名过滤，后续接入轻量ML模型
type SimpleImageQualityChecker struct {
	allowedExts map[string]bool
}

// NewSimpleImageQualityChecker 创建简单图片质量检测器
func NewSimpleImageQualityChecker() *SimpleImageQualityChecker {
	return &SimpleImageQualityChecker{
		allowedExts: map[string]bool{
			".jpg":  true,
			".jpeg": true,
			".png":  true,
			".bmp":  true,
		},
	}
}

// Filter 过滤不合格图片
func (q *SimpleImageQualityChecker) Filter(ctx context.Context, imageURLs []string) (valid []string, invalid []InvalidImage) {
	for _, url := range imageURLs {
		if !q.isValidImage(url) {
			invalid = append(invalid, InvalidImage{
				ImageURL: url,
				Reason:   "unsupported image format",
			})
			continue
		}
		valid = append(valid, url)
	}

	if valid == nil {
		valid = []string{}
	}
	if invalid == nil {
		invalid = []InvalidImage{}
	}
	return
}

// isValidImage 检查图片是否有效（基础检查：格式、URL格式）
func (q *SimpleImageQualityChecker) isValidImage(url string) bool {
	// 检查URL不为空
	if url == "" {
		return false
	}

	// 从URL中提取文件扩展名
	lowerURL := strings.ToLower(url)

	// 去除查询参数
	if idx := strings.Index(lowerURL, "?"); idx > 0 {
		lowerURL = lowerURL[:idx]
	}

	ext := strings.ToLower(filepath.Ext(lowerURL))
	if ext == "" {
		// 如果没有扩展名，默认允许（可能是MinIO生成的URL无扩展名）
		return true
	}

	return q.allowedExts[ext]
}

package ocr

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/edaptix/server/internal/config"
	"go.uber.org/zap"
)

// BaiduOCRClient 百度OCR云端客户端
type BaiduOCRClient struct {
	apiKey      string
	secretKey   string
	accessToken string
	tokenExpiry time.Time
	mu          sync.RWMutex
	httpClient  *http.Client
}

// NewBaiduOCRClient 创建百度OCR客户端
func NewBaiduOCRClient(cfg config.OCRConfig) *BaiduOCRClient {
	return &BaiduOCRClient{
		apiKey:    cfg.BaiduAPIKey,
		secretKey: cfg.BaiduSecretKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// getAccessToken 获取百度API Access Token（带缓存和自动刷新）
func (c *BaiduOCRClient) getAccessToken(ctx context.Context) (string, error) {
	c.mu.RLock()
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		token := c.accessToken
		c.mu.RUnlock()
		return token, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.accessToken, nil
	}

	url := fmt.Sprintf(
		"https://aip.baidubce.com/oauth/2.0/token?grant_type=client_credentials&client_id=%s&client_secret=%s",
		c.apiKey, c.secretKey,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", fmt.Errorf("create token request failed: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request access token failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read token response failed: %w", err)
	}

	var tokenResp BaiduAccessTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("parse token response failed: %w", err)
	}

	if tokenResp.Error != "" {
		return "", fmt.Errorf("baidu auth error: %s - %s", tokenResp.Error, tokenResp.ErrorDesc)
	}

	c.accessToken = tokenResp.AccessToken
	// 提前5分钟过期，避免边界问题
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-300) * time.Second)

	zap.L().Info("baidu OCR access token refreshed",
		zap.Time("expiry", c.tokenExpiry),
	)

	return c.accessToken, nil
}

// Recognize 通用文字识别（图片base64列表）
func (c *BaiduOCRClient) Recognize(ctx context.Context, images []string, mode OCRMode) (*OCRResult, error) {
	var allWords []OCRWordItem
	var textBuilder strings.Builder

	for i, img := range images {
		// 判断是URL还是base64
		var imageBase64 string
		if strings.HasPrefix(img, "http") {
			// 下载图片后转base64
			data, err := c.downloadImage(ctx, img)
			if err != nil {
				zap.L().Warn("download image failed, skipping",
					zap.String("url", img),
					zap.Error(err),
				)
				continue
			}
			imageBase64 = base64.StdEncoding.EncodeToString(data)
		} else {
			imageBase64 = img
		}

		words, err := c.recognizeGeneralBasic(ctx, imageBase64)
		if err != nil {
			zap.L().Warn("OCR recognize failed for image",
				zap.Int("index", i),
				zap.Error(err),
			)
			continue
		}

		allWords = append(allWords, words...)
		for _, w := range words {
			textBuilder.WriteString(w.Text)
			textBuilder.WriteString("\n")
		}
	}

	return &OCRResult{
		Text:   strings.TrimSpace(textBuilder.String()),
		Words:  allWords,
		Engine: "baidu-ocr",
	}, nil
}

// RecognizeWithURL 通过图片URL识别
func (c *BaiduOCRClient) RecognizeWithURL(ctx context.Context, imageURLs []string, mode OCRMode) (*OCRResult, error) {
	return c.Recognize(ctx, imageURLs, mode)
}

// RecognizeHandwriting 手写体识别
func (c *BaiduOCRClient) RecognizeHandwriting(ctx context.Context, imageBase64List []string) (*OCRResult, error) {
	var allWords []OCRWordItem
	var textBuilder strings.Builder

	for i, img := range imageBase64List {
		words, err := c.recognizeHandwriting(ctx, img)
		if err != nil {
			zap.L().Warn("handwriting OCR failed for image",
				zap.Int("index", i),
				zap.Error(err),
			)
			continue
		}

		allWords = append(allWords, words...)
		for _, w := range words {
			textBuilder.WriteString(w.Text)
			textBuilder.WriteString("\n")
		}
	}

	return &OCRResult{
		Text:   strings.TrimSpace(textBuilder.String()),
		Words:  allWords,
		Engine: "baidu-ocr-handwriting",
	}, nil
}

// RecognizeTable 表格识别
func (c *BaiduOCRClient) RecognizeTable(ctx context.Context, imageBase64 string) (*OCRResult, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get access token failed: %w", err)
	}

	url := fmt.Sprintf("https://aip.baidubce.com/rest/2.0/ocr/v1/table?access_token=%s", token)

	payload := fmt.Sprintf("image=%s", urlEncode(imageBase64))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBufferString(payload))
	if err != nil {
		return nil, fmt.Errorf("create table OCR request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("table OCR request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read table OCR response failed: %w", err)
	}

	var tableResp BaiduTableOCRResponse
	if err := json.Unmarshal(body, &tableResp); err != nil {
		return nil, fmt.Errorf("parse table OCR response failed: %w", err)
	}

	if tableResp.ErrorCode != 0 {
		return nil, fmt.Errorf("baidu table OCR error: %d - %s", tableResp.ErrorCode, tableResp.ErrorMsg)
	}

	var textBuilder strings.Builder
	for _, table := range tableResp.TablesResult {
		for _, cell := range table.Body {
			textBuilder.WriteString(cell.Words)
			textBuilder.WriteString("\t")
		}
		textBuilder.WriteString("\n")
	}

	return &OCRResult{
		Text:   strings.TrimSpace(textBuilder.String()),
		Engine: "baidu-ocr-table",
	}, nil
}

// recognizeGeneralBasic 通用文字识别（标准版）
func (c *BaiduOCRClient) recognizeGeneralBasic(ctx context.Context, imageBase64 string) ([]OCRWordItem, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get access token failed: %w", err)
	}

	url := fmt.Sprintf("https://aip.baidubce.com/rest/2.0/ocr/v1/general_basic?access_token=%s", token)

	payload := fmt.Sprintf("image=%s&detect_direction=true&paragraph=false&probability=true",
		urlEncode(imageBase64))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBufferString(payload))
	if err != nil {
		return nil, fmt.Errorf("create OCR request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OCR request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read OCR response failed: %w", err)
	}

	var ocrResp BaiduOCRResponse
	if err := json.Unmarshal(body, &ocrResp); err != nil {
		return nil, fmt.Errorf("parse OCR response failed: %w", err)
	}

	if ocrResp.ErrorCode != 0 {
		return nil, fmt.Errorf("baidu OCR error: %d - %s", ocrResp.ErrorCode, ocrResp.ErrorMsg)
	}

	var words []OCRWordItem
	for _, w := range ocrResp.WordsResult {
		word := OCRWordItem{
			Text:       w.Words,
			Confidence: w.Probability.Average,
			Location: &Location{
				Left:   w.Location.Left,
				Top:    w.Location.Top,
				Width:  w.Location.Width,
				Height: w.Location.Height,
			},
		}
		words = append(words, word)
	}

	return words, nil
}

// recognizeHandwriting 手写体识别
func (c *BaiduOCRClient) recognizeHandwriting(ctx context.Context, imageBase64 string) ([]OCRWordItem, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get access token failed: %w", err)
	}

	url := fmt.Sprintf("https://aip.baidubce.com/rest/2.0/ocr/v1/handwriting?access_token=%s", token)

	payload := fmt.Sprintf("image=%s&detect_direction=true&probability=true",
		urlEncode(imageBase64))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBufferString(payload))
	if err != nil {
		return nil, fmt.Errorf("create handwriting OCR request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("handwriting OCR request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read handwriting OCR response failed: %w", err)
	}

	var ocrResp BaiduOCRResponse
	if err := json.Unmarshal(body, &ocrResp); err != nil {
		return nil, fmt.Errorf("parse handwriting OCR response failed: %w", err)
	}

	if ocrResp.ErrorCode != 0 {
		return nil, fmt.Errorf("baidu handwriting OCR error: %d - %s", ocrResp.ErrorCode, ocrResp.ErrorMsg)
	}

	var words []OCRWordItem
	for _, w := range ocrResp.WordsResult {
		word := OCRWordItem{
			Text:       w.Words,
			Confidence: w.Probability.Average,
			Location: &Location{
				Left:   w.Location.Left,
				Top:    w.Location.Top,
				Width:  w.Location.Width,
				Height: w.Location.Height,
			},
		}
		words = append(words, word)
	}

	return words, nil
}

// downloadImage 下载图片到内存
func (c *BaiduOCRClient) downloadImage(ctx context.Context, imageURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create download request failed: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download image failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download image returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read image data failed: %w", err)
	}

	// 限制图片大小最大10MB
	if len(data) > 10*1024*1024 {
		return nil, fmt.Errorf("image size exceeds 10MB limit")
	}

	return data, nil
}

// urlEncode 对base64字符串做URL编码（仅替换特殊字符）
func urlEncode(s string) string {
	s = strings.ReplaceAll(s, "+", "%2B")
	s = strings.ReplaceAll(s, "/", "%2F")
	s = strings.ReplaceAll(s, "=", "%3D")
	return s
}

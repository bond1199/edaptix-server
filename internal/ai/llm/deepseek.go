package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/edaptix/server/internal/config"
	"go.uber.org/zap"
)

// DeepSeekClient DeepSeek V4 API客户端
type DeepSeekClient struct {
	flashCfg   config.AIModelConfig
	proCfg     config.AIModelConfig
	httpClient *http.Client
}

// NewDeepSeekClient 创建DeepSeek客户端
func NewDeepSeekClient(cfg config.AIConfig) *DeepSeekClient {
	return &DeepSeekClient{
		flashCfg: cfg.Flash,
		proCfg:   cfg.Pro,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Chat 发送对话请求
func (c *DeepSeekClient) Chat(ctx context.Context, model ModelType, messages []Message, opts ...ChatOption) (*ChatResponse, error) {
	cfg := c.getModelConfig(model)

	req := &ChatRequest{
		Model:    cfg.ModelName,
		Messages: messages,
		MaxTokens: cfg.MaxTokens,
	}

	// 应用选项
	for _, opt := range opts {
		opt(req)
	}

	// 如果未设置MaxTokens，使用配置默认值
	if req.MaxTokens == 0 {
		req.MaxTokens = cfg.MaxTokens
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	url := fmt.Sprintf("%s/chat/completions", cfg.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cfg.APIKey))

	// 设置超时
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.Timeout)*time.Second)
	defer cancel()
	httpReq = httpReq.WithContext(timeoutCtx)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ChatResponse
		_ = json.Unmarshal(respBody, &errResp)
		apiErr := &APIError{
			StatusCode: resp.StatusCode,
		}
		if errResp.Error != nil {
			apiErr.Message = errResp.Error.Message
			apiErr.Type = errResp.Error.Type
			apiErr.Code = errResp.Error.Code
		}
		return nil, apiErr
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}

	zap.L().Debug("deepseek chat completed",
		zap.String("model", string(model)),
		zap.Int("prompt_tokens", chatResp.Usage.PromptTokens),
		zap.Int("completion_tokens", chatResp.Usage.CompletionTokens),
	)

	return &chatResp, nil
}

// ChatWithJSON 发送对话请求并解析JSON响应
func (c *DeepSeekClient) ChatWithJSON(ctx context.Context, model ModelType, messages []Message, result interface{}, opts ...ChatOption) error {
	resp, err := c.Chat(ctx, model, messages, opts...)
	if err != nil {
		return err
	}

	if len(resp.Choices) == 0 {
		return fmt.Errorf("no response choices returned")
	}

	content := resp.Choices[0].Message.Content

	// 尝试提取JSON部分（AI可能在JSON前后附加了说明文字）
	jsonStr := extractJSON(content)
	if jsonStr == "" {
		return fmt.Errorf("no valid JSON found in response: %s", truncate(content, 200))
	}

	if err := json.Unmarshal([]byte(jsonStr), result); err != nil {
		return fmt.Errorf("unmarshal JSON response failed: %w (content: %s)", err, truncate(jsonStr, 200))
	}

	return nil
}

// ParseCatalog 解析教材目录为知识树
func (c *DeepSeekClient) ParseCatalog(ctx context.Context, ocrText string, subject string, grade int) (*CatalogParseResult, error) {
	messages := []Message{
		{
			Role:    "system",
			Content: CatalogParseSystemPrompt,
		},
		{
			Role: "user",
			Content: fmt.Sprintf(`请解析以下教材目录OCR文本，提取知识树结构。

学科：%s
年级：%d

OCR文本内容：
%s`, subject, grade, ocrText),
		},
	}

	var result CatalogParseResult
	if err := c.ChatWithJSON(ctx, ModelFlash, messages, &result, WithTemperature(0.1)); err != nil {
		return nil, fmt.Errorf("parse catalog failed: %w", err)
	}

	return &result, nil
}

// GradeEssay 批改作文（使用Pro模型）
func (c *DeepSeekClient) GradeEssay(ctx context.Context, essayContent string, grade int, prompt string) (*GradingResult, error) {
	messages := []Message{
		{
			Role:    "system",
			Content: EssayGradingSystemPrompt,
		},
		{
			Role: "user",
			Content: fmt.Sprintf(`请批改以下学生作文。

年级：%d
题目：%s

作文内容：
%s`, grade, prompt, essayContent),
		},
	}

	var result GradingResult
	if err := c.ChatWithJSON(ctx, ModelPro, messages, &result, WithTemperature(0.3)); err != nil {
		return nil, fmt.Errorf("grade essay failed: %w", err)
	}

	return &result, nil
}

// GradeAnswer 批改答题（使用Flash模型）
func (c *DeepSeekClient) GradeAnswer(ctx context.Context, question, correctAnswer, studentAnswer string, subject string) (*GradingResult, error) {
	messages := []Message{
		{
			Role: "system",
			Content: `你是一位专业的K12教师，负责批改学生的答案。请根据题目和标准答案判断学生答案是否正确，并给出评分和反馈。

请以JSON格式输出：
{
  "total_score": 得分,
  "max_score": 满分,
  "is_correct": true/false,
  "feedback": "反馈说明"
}`,
		},
		{
			Role: "user",
			Content: fmt.Sprintf(`学科：%s

题目：%s

标准答案：%s

学生答案：%s`, subject, question, correctAnswer, studentAnswer),
		},
	}

	var result GradingResult
	if err := c.ChatWithJSON(ctx, ModelFlash, messages, &result, WithTemperature(0.2)); err != nil {
		return nil, fmt.Errorf("grade answer failed: %w", err)
	}

	return &result, nil
}

// GenerateQuestions AI出题（使用Flash模型）
func (c *DeepSeekClient) GenerateQuestions(ctx context.Context, knowledgePoint string, questionType string, difficulty int, subject string, grade int, count int) (string, error) {
	difficultyDesc := map[int]string{
		1: "基础（识记）",
		2: "简单（理解）",
		3: "中等（应用）",
		4: "较难（分析）",
		5: "困难（综合）",
	}

	diffStr := difficultyDesc[difficulty]
	if diffStr == "" {
		diffStr = "中等（应用）"
	}

	messages := []Message{
		{
			Role: "system",
			Content: `你是一位专业的K12教育出题专家。请根据知识点、题型和难度要求，生成高质量的题目。

请以JSON数组格式输出：
[
  {
    "question_type": "题型",
    "content": "题目内容",
    "options": {"A": "选项A", "B": "选项B", "C": "选项C", "D": "选项D"},
    "answer": "正确答案",
    "analysis": "解析",
    "difficulty": 难度等级
  }
]

注意：选择题必须有4个选项，填空题options为null，解答题options为null且answer为详细解答步骤。`,
		},
		{
			Role: "user",
			Content: fmt.Sprintf(`请生成%d道%s题目。

学科：%s
年级：%d
知识点：%s
题型：%s
难度：%s`, count, questionType, subject, grade, knowledgePoint, questionType, diffStr),
		},
	}

	resp, err := c.Chat(ctx, ModelFlash, messages, WithTemperature(0.7))
	if err != nil {
		return "", fmt.Errorf("generate questions failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response returned")
	}

	return resp.Choices[0].Message.Content, nil
}

// getModelConfig 获取模型配置
func (c *DeepSeekClient) getModelConfig(model ModelType) config.AIModelConfig {
	switch model {
	case ModelPro:
		return c.proCfg
	default:
		return c.flashCfg
	}
}

// ChatOption 对话选项
type ChatOption func(*ChatRequest)

// WithTemperature 设置温度
func WithTemperature(temp float64) ChatOption {
	return func(req *ChatRequest) {
		req.Temperature = temp
	}
}

// WithMaxTokens 设置最大token数
func WithMaxTokens(max int) ChatOption {
	return func(req *ChatRequest) {
		req.MaxTokens = max
	}
}

// WithTopP 设置top_p
func WithTopP(topP float64) ChatOption {
	return func(req *ChatRequest) {
		req.TopP = topP
	}
}

// extractJSON 从文本中提取JSON字符串
func extractJSON(text string) string {
	// 尝试直接解析
	trimmed := trimWhitespace(text)
	if isJSON(trimmed) {
		return trimmed
	}

	// 尝试提取 ```json ... ``` 代码块
	if start := indexOf(trimmed, "```json"); start >= 0 {
		start += 7 // len("```json")
		end := indexOf(trimmed, "```")
		if end > start {
			jsonStr := trimWhitespace(trimmed[start:end])
			if isJSON(jsonStr) {
				return jsonStr
			}
		}
	}

	// 尝试提取 ``` ... ``` 代码块
	if start := indexOf(trimmed, "```"); start >= 0 {
		start += 3
		end := indexOf(trimmed[start:], "```")
		if end >= 0 {
			jsonStr := trimWhitespace(trimmed[start : start+end])
			if isJSON(jsonStr) {
				return jsonStr
			}
		}
	}

	// 尝试提取 { ... } 或 [ ... ]
	for _, open := range []byte{'{', '['} {
		start := bytes.IndexByte([]byte(trimmed), open)
		if start >= 0 {
			close := byte('}')
			if open == '[' {
				close = ']'
			}
			// 找到最后一个匹配的闭合括号
			end := lastIndexOfByte(trimmed, close)
			if end > start {
				jsonStr := trimmed[start : end+1]
				if isJSON(jsonStr) {
					return jsonStr
				}
			}
		}
	}

	return ""
}

func isJSON(s string) bool {
	s = trimWhitespace(s)
	return (len(s) >= 2 && ((s[0] == '{' && s[len(s)-1] == '}') || (s[0] == '[' && s[len(s)-1] == ']')))
}

func trimWhitespace(s string) string {
	// 简单trim
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\n' || s[0] == '\r') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t' || s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func lastIndexOfByte(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

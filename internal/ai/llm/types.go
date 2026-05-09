package llm

import (
	"fmt"
)

// ModelType DeepSeek模型类型
type ModelType string

const (
	ModelFlash ModelType = "flash" // DeepSeek V4-Flash（快速，主用）
	ModelPro   ModelType = "pro"   // DeepSeek V4-Pro（深度推理，辅用）
)

// Message LLM对话消息
type Message struct {
	Role    string `json:"role"`    // system, user, assistant
	Content string `json:"content"`
}

// ChatRequest LLM对话请求
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	TopP        float64   `json:"top_p,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

// ChatResponse LLM对话响应
type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Error   *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// CatalogParseResult 目录解析结果（AI返回的结构化知识树）
type CatalogParseResult struct {
	Subject  string          `json:"subject"`
	Grade    int             `json:"grade"`
	Edition  string          `json:"edition"`
	Chapters []ParsedChapter `json:"chapters"`
}

// ParsedChapter 解析的章
type ParsedChapter struct {
	Name      string          `json:"name"`
	Sections  []ParsedSection `json:"sections,omitempty"`
	SortOrder int             `json:"sort_order"`
}

// ParsedSection 解析的节
type ParsedSection struct {
	Name             string   `json:"name"`
	KnowledgePoints  []string `json:"knowledge_points,omitempty"`
	SortOrder        int      `json:"sort_order"`
}

// GradingResult 批改结果
type GradingResult struct {
	TotalScore  float64         `json:"total_score"`
	MaxScore    float64         `json:"max_score"`
	IsCorrect   bool            `json:"is_correct"`
	Feedback    string          `json:"feedback"`
	ItemResults []ItemGrading   `json:"item_results,omitempty"`
}

// ItemGrading 单题批改
type ItemGrading struct {
	QuestionIndex int     `json:"question_index"`
	Score         float64 `json:"score"`
	MaxScore      float64 `json:"max_score"`
	IsCorrect     bool    `json:"is_correct"`
	Feedback      string  `json:"feedback"`
}

// APIError DeepSeek API错误
type APIError struct {
	StatusCode int
	Message    string
	Type       string
	Code       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("deepseek api error: status=%d, type=%s, code=%s, message=%s",
		e.StatusCode, e.Type, e.Code, e.Message)
}

// DeepSeekStreamChunk 流式响应块（预留，当前不用）
type DeepSeekStreamChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// CatalogParsePrompt 用于目录解析的系统提示词
const CatalogParseSystemPrompt = `你是一个K12教育领域的知识树解析专家。你的任务是从教材目录的OCR文本中，提取出结构化的知识树。

请严格按照以下JSON格式输出，不要输出任何其他内容：
{
  "subject": "学科名称",
  "grade": 年级数字,
  "edition": "教材版本",
  "chapters": [
    {
      "name": "章节名称",
      "sort_order": 1,
      "sections": [
        {
          "name": "小节名称",
          "sort_order": 1,
          "knowledge_points": ["知识点1", "知识点2"]
        }
      ]
    }
  ]
}

注意：
1. 年级用数字1-12表示
2. 章节顺序按教材目录顺序排列
3. 知识点要尽可能细化到最小可测评单元
4. 如果无法识别年级和版本，请根据内容推测
5. 只输出JSON，不要输出任何解释文字`

// EssayGradingPrompt 作文批改系统提示词
const EssayGradingSystemPrompt = `你是一位资深的K12语文教师，擅长批改学生作文。请根据以下维度评分：
1. 内容（30分）：主题明确，内容充实，有真情实感
2. 结构（20分）：层次清晰，过渡自然，首尾呼应
3. 语言（30分）：用词准确，句式多样，修辞恰当
4. 书写（20分）：书写规范，标点正确，格式规范

请以JSON格式输出批改结果：
{
  "total_score": 总分,
  "max_score": 100,
  "is_correct": true/false（是否及格，60分以上），
  "feedback": "详细评语，包含优点和改进建议"
}`

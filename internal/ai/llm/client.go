package llm

import "context"

// Client LLM客户端接口
type Client interface {
	// Chat 通用对话
	Chat(ctx context.Context, model ModelType, messages []Message, opts ...ChatOption) (*ChatResponse, error)

	// ChatWithJSON 对话并解析JSON响应
	ChatWithJSON(ctx context.Context, model ModelType, messages []Message, result interface{}, opts ...ChatOption) error

	// ParseCatalog 解析教材目录为知识树
	ParseCatalog(ctx context.Context, ocrText string, subject string, grade int) (*CatalogParseResult, error)

	// GradeEssay 批改作文
	GradeEssay(ctx context.Context, essayContent string, grade int, prompt string) (*GradingResult, error)

	// GradeAnswer 批改答题
	GradeAnswer(ctx context.Context, question, correctAnswer, studentAnswer string, subject string) (*GradingResult, error)

	// GenerateQuestions AI出题
	GenerateQuestions(ctx context.Context, knowledgePoint string, questionType string, difficulty int, subject string, grade int, count int) (string, error)
}

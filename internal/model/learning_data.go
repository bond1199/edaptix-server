package model

import (
	"encoding/json"
	"time"
)

// LearningUpload 上传批次表
type LearningUpload struct {
	ID         int64          `gorm:"primaryKey" json:"id"`
	UserID     int64          `gorm:"not null;index" json:"user_id"`
	UploadType string         `gorm:"type:varchar(20);not null" json:"upload_type"` // catalog/homework/exam/answer_sheet
	Source     string         `gorm:"type:varchar(20);not null" json:"source"`       // camera/album
	Subject    string         `gorm:"type:varchar(30);not null;default:''" json:"subject"`
	Status     int16          `gorm:"type:smallint;not null;default:1;index" json:"status"` // 1:待处理 2:AI处理中 3:已完成 4:处理失败
	PageCount  int            `gorm:"not null;default:0" json:"page_count"`
	CreatedAt  time.Time      `json:"created_at"`
}

func (LearningUpload) TableName() string {
	return "learning_uploads"
}

// UploadItem 上传素材明细表
type UploadItem struct {
	ID            int64          `gorm:"primaryKey" json:"id"`
	UploadID      int64          `gorm:"not null;index" json:"upload_id"`
	ImageURL      string         `gorm:"type:varchar(500);not null" json:"image_url"`
	PageIndex     int            `gorm:"not null;default:0" json:"page_index"`
	IsValid       bool           `gorm:"not null;default:true" json:"is_valid"`
	InvalidReason string         `gorm:"type:varchar(50)" json:"invalid_reason"`
	OCRResult     json.RawMessage `gorm:"type:jsonb" json:"ocr_result"`
	CreatedAt     time.Time      `json:"created_at"`
}

func (UploadItem) TableName() string {
	return "upload_items"
}

// ErrorQuestion 错题表
type ErrorQuestion struct {
	ID               int64     `gorm:"primaryKey" json:"id"`
	UserID           int64     `gorm:"not null;index" json:"user_id"`
	Subject          string    `gorm:"type:varchar(30);not null;index:idx_eq_subject" json:"subject"`
	KnowledgeNodeID  *int64    `gorm:"index" json:"knowledge_node_id"`
	QuestionType     string    `gorm:"type:varchar(30);not null" json:"question_type"`
	QuestionContent  string    `gorm:"type:text;not null" json:"question_content"`
	CorrectAnswer    string    `gorm:"type:text" json:"correct_answer"`
	StudentAnswer    string    `gorm:"type:text" json:"student_answer"`
	ErrorType        string    `gorm:"type:varchar(20);not null" json:"error_type"` // wrong/blank/guessed
	SourceType       string    `gorm:"type:varchar(20);not null" json:"source_type"` // homework/exam/daily_task
	SourceID         *int64    `json:"source_id"`
	Difficulty       int16     `gorm:"type:smallint;not null;default:1" json:"difficulty"`
	ReviewCount      int       `gorm:"not null;default:0" json:"review_count"`
	LastReviewed     *time.Time `json:"last_reviewed"`
	IsResolved       bool      `gorm:"not null;default:false;index" json:"is_resolved"`
	CreatedAt        time.Time `json:"created_at"`
}

func (ErrorQuestion) TableName() string {
	return "error_questions"
}

// QuestionBank 题库表
type QuestionBank struct {
	ID              int64           `gorm:"primaryKey" json:"id"`
	Subject         string          `gorm:"type:varchar(30);not null" json:"subject"`
	Grade           int16           `gorm:"not null" json:"grade"`
	KnowledgeNodeID *int64          `gorm:"index" json:"knowledge_node_id"`
	QuestionType    string          `gorm:"type:varchar(30);not null" json:"question_type"`
	Difficulty      int16           `gorm:"type:smallint;not null;default:1" json:"difficulty"`
	Content         string          `gorm:"type:text;not null" json:"content"`
	Options         json.RawMessage `gorm:"type:jsonb" json:"options"`
	Answer          string          `gorm:"type:text;not null" json:"answer"`
	Analysis        string          `gorm:"type:text" json:"analysis"`
	Source          string          `gorm:"type:varchar(30);not null;default:'ai'" json:"source"`
	ExamFrequency   int16           `gorm:"type:smallint;not null;default:1;index" json:"exam_frequency"`
	IsValid         bool            `gorm:"not null;default:true" json:"is_valid"`
	UsageCount      int             `gorm:"not null;default:0" json:"usage_count"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

func (QuestionBank) TableName() string {
	return "question_bank"
}

// UserQuestionHistory 用户已做题目记录
type UserQuestionHistory struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	UserID      int64     `gorm:"not null;index" json:"user_id"`
	QuestionID  int64     `gorm:"not null;index" json:"question_id"`
	TaskItemID  *int64    `json:"task_item_id"`
	IsCorrect   *bool     `json:"is_correct"`
	AnsweredAt  time.Time `gorm:"not null;default:now()" json:"answered_at"`
}

func (UserQuestionHistory) TableName() string {
	return "user_question_history"
}

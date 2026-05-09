package model

import (
	"time"

	"gorm.io/gorm"
)

// LearningUpload 学习数据上传批次
type LearningUpload struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	StudentID   uint           `gorm:"index;not null" json:"student_id"`
	Subject     string         `gorm:"type:varchar(20);not null" json:"subject"`
	SourceType  string         `gorm:"type:varchar(30);not null" json:"source_type"` // photo, pdf, manual
	Status      string         `gorm:"type:varchar(20);not null;default:pending" json:"status"` // pending, processing, completed, failed
	ItemCount   int            `gorm:"default:0" json:"item_count"`
	OCRResult   string         `gorm:"type:text" json:"ocr_result"`     // OCR原始结果JSON
	AIResult    string         `gorm:"type:text" json:"ai_result"`      // AI解析结果JSON
	ErrorMsg    string         `gorm:"type:text" json:"error_msg"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Student User        `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	Items   []UploadItem `gorm:"foreignKey:UploadID" json:"items,omitempty"`
}

func (LearningUpload) TableName() string {
	return "learning_uploads"
}

// UploadItem 上传项（单张图片/PDF页）
type UploadItem struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	UploadID     uint           `gorm:"index;not null" json:"upload_id"`
	FileURL      string         `gorm:"type:varchar(500);not null" json:"file_url"`
	FileType     string         `gorm:"type:varchar(10)" json:"file_type"` // jpg, png, pdf
	PageNum      int            `gorm:"default:0" json:"page_num"`         // PDF页码
	OCRText      string         `gorm:"type:text" json:"ocr_text"`
	OCRConfidence float64       `json:"ocr_confidence"`
	Status       string         `gorm:"type:varchar(20);not null;default:pending" json:"status"` // pending, processed, failed
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	Upload LearningUpload `gorm:"foreignKey:UploadID" json:"-"`
}

func (UploadItem) TableName() string {
	return "upload_items"
}

// ErrorQuestion 错题记录
type ErrorQuestion struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	StudentID      uint           `gorm:"index;not null" json:"student_id"`
	UploadID       uint           `gorm:"index" json:"upload_id"`
	Subject        string         `gorm:"type:varchar(20);not null; json:"subject"`
	KnowledgeNodeID *uint         `gorm:"index" json:"knowledge_node_id"`                 // 关联知识点
	QuestionText   string         `gorm:"type:text;not null" json:"question_text"`        // 题目内容
	QuestionImage  string         `gorm:"type:varchar(500)" json:"question_image"`        // 题目图片URL
	StudentAnswer  string         `gorm:"type:text" json:"student_answer"`                // 学生作答
	CorrectAnswer  string         `gorm:"type:text" json:"correct_answer"`                // 正确答案
	ErrorType      string         `gorm:"type:varchar(30)" json:"error_type"`             // 计算错误, 概念混淆, 审题不清, 方法不当
	Difficulty     int            `gorm:"type:smallint;default:3" json:"difficulty"`      // 难度1-5
	SourceExam     string         `gorm:"type:varchar(100)" json:"source_exam"`           // 来源试卷
	Mastered       bool           `gorm:"default:false" json:"mastered"`                  // 是否已掌握
	ReviewCount    int            `gorm:"default:0" json:"review_count"`                  // 复习次数
	NextReviewAt   *time.Time     `json:"next_review_at"`                                  // 下次复习时间(间隔重复)
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`

	Student        User           `gorm:"foreignKey:StudentID" json:"-"`
	Upload         LearningUpload `gorm:"foreignKey:UploadID" json:"-"`
	KnowledgeNode  *KnowledgeNode `gorm:"foreignKey:KnowledgeNodeID" json:"knowledge_node,omitempty"`
}

func (ErrorQuestion) TableName() string {
	return "error_questions"
}

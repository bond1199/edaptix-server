package model

import (
	"time"

	"gorm.io/gorm"
)

// DailyTask 每日学习任务
type DailyTask struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	StudentID   uint           `gorm:"index;not null" json:"student_id"`
	TaskDate    time.Time      `gorm:"type:date;not null;index" json:"task_date"`
	Subject     string         `gorm:"type:varchar(20);not null" json:"subject"`
	Status      string         `gorm:"type:varchar(20);not null;default:pending" json:"status"` // pending, in_progress, completed, skipped
	TotalItems  int            `gorm:"default:0" json:"total_items"`
	DoneItems   int            `gorm:"default:0" json:"done_items"`
	TimeLimit   int            `json:"time_limit"`         // 建议用时(分钟)
	TimeSpent   int            `json:"time_spent"`         // 实际用时(分钟)
	Score       *float64       `json:"score"`              // 完成得分
	Feedback    string         `gorm:"type:text" json:"feedback"` // AI反馈
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Student User       `gorm:"foreignKey:StudentID" json:"-"`
	Items   []TaskItem `gorm:"foreignKey:TaskID" json:"items,omitempty"`
}

func (DailyTask) TableName() string {
	return "daily_tasks"
}

// TaskItem 任务项
type TaskItem struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	TaskID           uint           `gorm:"index;not null" json:"task_id"`
	ErrorQuestionID  *uint          `gorm:"index" json:"error_question_id"`     // 关联错题
	KnowledgeNodeID  *uint          `gorm:"index" json:"knowledge_node_id"`     // 关联知识点
	ItemType         string         `gorm:"type:varchar(30);not null" json:"item_type"` // review, practice, explain, quiz
	Content          string         `gorm:"type:text" json:"content"`           // 任务内容
	QuestionData     string         `gorm:"type:text" json:"question_data"`     // 题目数据JSON
	SortOrder        int            `gorm:"not null;default:0" json:"sort_order"`
	Status           string         `gorm:"type:varchar(20);not null;default:pending" json:"status"` // pending, done, skipped
	StudentAnswer    string         `gorm:"type:text" json:"student_answer"`
	IsCorrect        *bool          `json:"is_correct"`
	TimeSpent        int            `json:"time_spent"` // 该题用时(秒)
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`

	Task           DailyTask     `gorm:"foreignKey:TaskID" json:"-"`
	ErrorQuestion  *ErrorQuestion `gorm:"foreignKey:ErrorQuestionID" json:"error_question,omitempty"`
	KnowledgeNode  *KnowledgeNode `gorm:"foreignKey:KnowledgeNodeID" json:"knowledge_node,omitempty"`
}

func (TaskItem) TableName() string {
	return "task_items"
}

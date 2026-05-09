package model

import (
	"encoding/json"
	"time"
)

// DailyTask 每日任务表
type DailyTask struct {
	ID             int64     `gorm:"primaryKey" json:"id"`
	UserID         int64     `gorm:"not null;index:idx_dt_user_date" json:"user_id"`
	TaskDate       time.Time `gorm:"type:date;not null" json:"task_date"`
	Subject        string    `gorm:"type:varchar(30);not null;default:''" json:"subject"`
	TaskMode       string    `gorm:"type:varchar(10);not null" json:"task_mode"` // online/offline
	Status         int16     `gorm:"type:smallint;not null;default:1;index" json:"status"` // 1:待完成 2:进行中 3:已完成 4:已逾期 5:已作废
	TotalItems     int       `gorm:"not null;default:0" json:"total_items"`
	CompletedItems int       `gorm:"not null;default:0" json:"completed_items"`
	CorrectItems   int       `gorm:"not null;default:0" json:"correct_items"`
	TimeLimitMin   int       `gorm:"not null;default:0" json:"time_limit_min"`
	ActualTimeMin  *int      `json:"actual_time_min"`
	StartAt        *time.Time `json:"start_at"`
	FinishAt       *time.Time `json:"finish_at"`
	PDFUrl         string    `gorm:"type:varchar(500)" json:"pdf_url"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (DailyTask) TableName() string {
	return "daily_tasks"
}

// TaskItem 任务题目明细表
type TaskItem struct {
	ID               int64           `gorm:"primaryKey" json:"id"`
	TaskID           int64           `gorm:"not null;index" json:"task_id"`
	QuestionID       *int64          `json:"question_id"`
	KnowledgeNodeID  *int64          `gorm:"index" json:"knowledge_node_id"`
	QuestionType     string          `gorm:"type:varchar(30);not null" json:"question_type"`
	QuestionContent  string          `gorm:"type:text;not null" json:"question_content"`
	Options          json.RawMessage `gorm:"type:jsonb" json:"options"`
	CorrectAnswer    string          `gorm:"type:text;not null" json:"correct_answer"`
	Difficulty       int16           `gorm:"type:smallint;not null;default:1" json:"difficulty"`
	ItemMode         string          `gorm:"type:varchar(10);not null;default:'remedial'" json:"item_mode"` // remedial/advanced
	SortOrder        int             `gorm:"not null;default:0" json:"sort_order"`
	Status           int16           `gorm:"type:smallint;not null;default:1;index" json:"status"` // 1:待答 2:已答 3:已批改
	StudentAnswer    string          `gorm:"type:text" json:"student_answer"`
	IsCorrect        *bool           `json:"is_correct"`
	Score            *float64        `gorm:"type:decimal(5,2)" json:"score"`
	AnswerDuration   *int            `json:"answer_duration"`
	GradingResult    json.RawMessage `gorm:"type:jsonb" json:"grading_result"`
	CreatedAt        time.Time       `json:"created_at"`
}

func (TaskItem) TableName() string {
	return "task_items"
}

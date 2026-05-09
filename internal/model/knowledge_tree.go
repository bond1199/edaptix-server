package model

import (
	"time"
)

// KnowledgeTree 学科知识树
type KnowledgeTree struct {
	ID              int64     `gorm:"primaryKey" json:"id"`
	UserID          int64     `gorm:"not null;index" json:"user_id"`
	Subject         string    `gorm:"type:varchar(30);not null" json:"subject"`
	Grade           int16     `gorm:"not null" json:"grade"`
	TextbookEdition string    `gorm:"type:varchar(50)" json:"textbook_edition"`
	Status          int16     `gorm:"type:smallint;not null;default:1" json:"status"` // 1:正常 2:AI处理中 3:处理失败
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (KnowledgeTree) TableName() string {
	return "knowledge_trees"
}

// KnowledgeNode 知识节点（五级结构）
type KnowledgeNode struct {
	ID             int64     `gorm:"primaryKey" json:"id"`
	TreeID         int64     `gorm:"not null;index" json:"tree_id"`
	ParentID       *int64    `gorm:"index" json:"parent_id"`
	Level          int16     `gorm:"type:smallint;not null" json:"level"`   // 1:年级 2:科目 3:章节 4:小节 5:知识点
	Name           string    `gorm:"type:varchar(200);not null" json:"name"`
	SortOrder      int       `gorm:"not null;default:0" json:"sort_order"`
	MasteryRate    float64   `gorm:"type:decimal(5,2);not null;default:0" json:"mastery_rate"`
	QuestionCount  int       `gorm:"not null;default:0" json:"question_count"`
	CorrectCount   int       `gorm:"not null;default:0" json:"correct_count"`
	ErrorCount     int       `gorm:"not null;default:0" json:"error_count"`
	LastPracticed  *time.Time `json:"last_practiced"`
	IsLocked       bool      `gorm:"not null;default:false" json:"is_locked"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (KnowledgeNode) TableName() string {
	return "knowledge_nodes"
}

package model

import (
	"time"

	"gorm.io/gorm"
)

// KnowledgeTree 知识树（按学科+年级+教材版本组织）
type KnowledgeTree struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	Subject       string         `gorm:"type:varchar(20);not null" json:"subject"`     // 学科: math, physics, chemistry, english
	Grade         string         `gorm:"type:varchar(20);not null" json:"grade"`       // 年级
	TextbookVer   string         `gorm:"type:varchar(50)" json:"textbook_ver"`         // 教材版本: 人教版, 北师大版等
	Name          string         `gorm:"type:varchar(100);not null" json:"name"`       // 知识树名称
	Description   string         `gorm:"type:text" json:"description"`
	Version       int            `gorm:"not null;default:1" json:"version"`            // 版本号，支持知识树迭代
	Status        int            `gorm:"type:smallint;not null;default:1" json:"status"` // 1:启用 0:禁用
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

	Nodes []KnowledgeNode `gorm:"foreignKey:TreeID" json:"nodes,omitempty"`
}

func (KnowledgeTree) TableName() string {
	return "knowledge_trees"
}

// KnowledgeNode 知识节点
type KnowledgeNode struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	TreeID      uint           `gorm:"index;not null" json:"tree_id"`
	ParentID    *uint          `gorm:"index" json:"parent_id"`                        // 父节点ID，根节点为nil
	Code        string         `gorm:"type:varchar(50);index" json:"code"`            // 节点编码: 如 M-G1-C1-S2
	Name        string         `gorm:"type:varchar(100);not null" json:"name"`        // 节点名称
	Level       int            `gorm:"not null" json:"level"`                         // 层级: 1章 2节 3知识点 4子知识点
	SortOrder   int            `gorm:"not null;default:0" json:"sort_order"`          // 同级排序
	Description string         `gorm:"type:text" json:"description"`
	Keywords    string         `gorm:"type:text" json:"keywords"`                     // 关键词，JSON数组格式
	Difficulty  int            `gorm:"type:smallint;default:3" json:"difficulty"`     // 难度等级1-5
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Tree   KnowledgeTree `gorm:"foreignKey:TreeID" json:"-"`
	Parent *KnowledgeNode `gorm:"foreignKey:ParentID" json:"-"`
	Children []KnowledgeNode `gorm:"foreignKey:ParentID" json:"children,omitempty"`
}

func (KnowledgeNode) TableName() string {
	return "knowledge_nodes"
}

package model

import (
	"time"

	"gorm.io/gorm"
)

// RiskRecord 风控记录
type RiskRecord struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	StudentID   uint           `gorm:"index;not null" json:"student_id"`
	TaskID      *uint          `gorm:"index" json:"task_id"`
	RiskType    string         `gorm:"type:varchar(30);not null;index" json:"risk_type"` // cheating, proxy, abnormal_speed, copy_paste
	Severity    string         `gorm:"type:varchar(10);not null;default:low" json:"severity"` // low, medium, high
	Detail      string         `gorm:"type:text" json:"detail"`          // 风控详情JSON
	Evidence    string         `gorm:"type:text" json:"evidence"`        // 证据数据JSON
	Handled     bool           `gorm:"default:false" json:"handled"`
	HandlerID   *uint          `json:"handler_id"`                       // 处理人ID
	HandleNote  string         `gorm:"type:text" json:"handle_note"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Student User       `gorm:"foreignKey:StudentID" json:"-"`
	Task    *DailyTask `gorm:"foreignKey:TaskID" json:"task,omitempty"`
}

func (RiskRecord) TableName() string {
	return "risk_records"
}

// IntegrityProfile 学业诚信档案
type IntegrityProfile struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	StudentID          uint           `gorm:"uniqueIndex;not null" json:"student_id"`
	TrustScore         float64        `gorm:"default:100" json:"trust_score"`               // 信任分0-100
	RiskCount          int            `gorm:"default:0" json:"risk_count"`                  // 风控触发次数
	HighRiskCount      int            `gorm:"default:0" json:"high_risk_count"`             // 高危触发次数
	LastRiskAt         *time.Time     `json:"last_risk_at"`
	PenaltyEndAt       *time.Time     `json:"penalty_end_at"`                               // 惩罚结束时间
	BehaviorSummary    string         `gorm:"type:text" json:"behavior_summary"`            // 行为摘要JSON
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`

	Student User `gorm:"foreignKey:StudentID" json:"-"`
}

func (IntegrityProfile) TableName() string {
	return "integrity_profiles"
}

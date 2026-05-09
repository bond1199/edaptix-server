package model

import (
	"time"

	"gorm.io/gorm"
)

// User 用户基础表
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Phone     string         `gorm:"type:varchar(20);uniqueIndex;not null" json:"phone"`
	Password  string         `gorm:"type:varchar(255);not null" json:"-"`
	Name      string         `gorm:"type:varchar(50)" json:"name"`
	Avatar    string         `gorm:"type:varchar(500)" json:"avatar"`
	Role      string         `gorm:"type:varchar(20);not null;default:student" json:"role"` // student, parent, teacher, admin
	Status    int            `gorm:"type:smallint;not null;default:1" json:"status"`        // 1:正常 0:禁用
	LastLogin *time.Time     `json:"last_login"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string {
	return "users"
}

// StudentProfile 学生档案
type StudentProfile struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	UserID         uint           `gorm:"uniqueIndex;not null" json:"user_id"`
	Grade          string         `gorm:"type:varchar(20)" json:"grade"`        // 年级
	SubjectScope   string         `gorm:"type:varchar(200)" json:"subject_scope"` // 关注学科,逗号分隔
	SchoolName     string         `gorm:"type:varchar(100)" json:"school_name"`
	TargetScore    int            `json:"target_score"`                         // 目标分数
	StudyGoal      string         `gorm:"type:varchar(500)" json:"study_goal"`   // 学习目标描述
	LearningStyle  string         `gorm:"type:varchar(50)" json:"learning_style"` // 学习风格
	DailyMinutes   int            `gorm:"default:30" json:"daily_minutes"`       // 每日可用学习时长(分钟)
	AIFeedback     string         `gorm:"type:text" json:"ai_feedback"`          // AI生成的学情反馈
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (StudentProfile) TableName() string {
	return "student_profiles"
}

// ParentAccount 家长账户
type ParentAccount struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"uniqueIndex;not null" json:"user_id"`
	RealName  string         `gorm:"type:varchar(50)" json:"real_name"`
	Phone     string         `gorm:"type:varchar(20)" json:"phone"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (ParentAccount) TableName() string {
	return "parent_accounts"
}

// ParentStudentBinding 家长-学生绑定关系
type ParentStudentBinding struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	ParentID    uint           `gorm:"index;not null" json:"parent_id"`
	StudentID   uint           `gorm:"index;not null" json:"student_id"`
	Relation    string         `gorm:"type:varchar(20)" json:"relation"` // father, mother, guardian
	Status      int            `gorm:"type:smallint;not null;default:1" json:"status"` // 1:已绑定 0:已解绑
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Parent  ParentAccount `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Student User          `gorm:"foreignKey:StudentID" json:"student,omitempty"`
}

func (ParentStudentBinding) TableName() string {
	return "parent_student_bindings"
}

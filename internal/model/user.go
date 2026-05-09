package model

import (
	"time"

	"gorm.io/gorm"
)

// User 用户基础表
type User struct {
	ID            int64          `gorm:"primaryKey" json:"id"`
	Phone         string         `gorm:"type:varchar(20);uniqueIndex;not null" json:"phone"`
	PasswordHash  string         `gorm:"column:password_hash;type:varchar(255);not null" json:"-"`
	Role          string         `gorm:"type:varchar(20);not null;default:student" json:"role"` // student / parent
	Status        int16          `gorm:"type:smallint;not null;default:1" json:"status"`        // 1:正常 2:禁用 3:冻结
	Initialized   bool           `gorm:"not null;default:false" json:"initialized"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string {
	return "users"
}

// StudentProfile 学生档案
type StudentProfile struct {
	ID          int64       `gorm:"primaryKey" json:"id"`
	UserID      int64       `gorm:"uniqueIndex;not null" json:"user_id"`
	RealName    string      `gorm:"type:varchar(50)" json:"real_name"`
	Grade       int16       `gorm:"not null" json:"grade"`            // 年级 1-12
	GradeStage  string      `gorm:"type:varchar(10);not null" json:"grade_stage"` // primary / junior / senior
	SchoolName  string      `gorm:"type:varchar(100)" json:"school_name"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (StudentProfile) TableName() string {
	return "student_profiles"
}

// ParentAccount 家长账号
type ParentAccount struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	UserID    int64     `gorm:"uniqueIndex;not null" json:"user_id"`
	RealName  string    `gorm:"type:varchar(50)" json:"real_name"`
	CreatedAt time.Time `json:"created_at"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (ParentAccount) TableName() string {
	return "parent_accounts"
}

// ParentStudentBinding 家长-学生绑定
type ParentStudentBinding struct {
	ID         int64      `gorm:"primaryKey" json:"id"`
	ParentID   int64      `gorm:"not null;index" json:"parent_id"`
	StudentID  int64      `gorm:"not null;index" json:"student_id"`
	BindCode   string     `gorm:"type:varchar(10);uniqueIndex" json:"bind_code"`
	BindQrcode string     `gorm:"type:varchar(255)" json:"bind_qrcode"`
	Status     int16      `gorm:"type:smallint;not null;default:1" json:"status"` // 1:绑定中 2:已绑定 3:已解绑
	BoundAt    *time.Time `json:"bound_at"`
	CreatedAt  time.Time  `json:"created_at"`

	Parent  ParentAccount `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Student User          `gorm:"foreignKey:StudentID" json:"student,omitempty"`
}

func (ParentStudentBinding) TableName() string {
	return "parent_student_bindings"
}

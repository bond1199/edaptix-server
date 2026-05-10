package request

// GenerateTaskRequest 手动生成任务请求
type GenerateTaskRequest struct {
	Subject  string `json:"subject" binding:"required,oneof=语文 数学 英语 物理 化学 生物 历史 地理 政治"`
	TaskMode string `json:"task_mode" binding:"required,oneof=online offline"`
}

// SubmitAnswerRequest 提交答案请求
type SubmitAnswerRequest struct {
	Answer         string `json:"answer" binding:"required"`
	AnswerDuration *int   `json:"answer_duration"` // 答题时长（秒）
}

// TaskQueryRequest 任务查询请求
type TaskQueryRequest struct {
	Subject  string `form:"subject"`
	Status   *int16 `form:"status"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

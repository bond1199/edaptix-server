package response

import "encoding/json"

// TaskListResponse 任务列表项
type TaskListResponse struct {
	ID             int64  `json:"id"`
	Subject        string `json:"subject"`
	TaskDate       string `json:"task_date"`
	TaskMode       string `json:"task_mode"`
	Status         int16  `json:"status"`
	TotalItems     int    `json:"total_items"`
	CompletedItems int    `json:"completed_items"`
	CorrectItems   int    `json:"correct_items"`
	TimeLimitMin   int    `json:"time_limit_min"`
	CreatedAt      string `json:"created_at"`
}

// TaskDetailResponse 任务详情
type TaskDetailResponse struct {
	ID             int64               `json:"id"`
	Subject        string              `json:"subject"`
	TaskDate       string              `json:"task_date"`
	TaskMode       string              `json:"task_mode"`
	Status         int16               `json:"status"`
	TotalItems     int                 `json:"total_items"`
	CompletedItems int                 `json:"completed_items"`
	CorrectItems   int                 `json:"correct_items"`
	TimeLimitMin   int                 `json:"time_limit_min"`
	ActualTimeMin  *int                `json:"actual_time_min,omitempty"`
	PDFUrl         string              `json:"pdf_url,omitempty"`
	Items          []TaskItemResponse  `json:"items,omitempty"`
	CreatedAt      string              `json:"created_at"`
}

// TaskItemResponse 任务题目项
type TaskItemResponse struct {
	ID              int64            `json:"id"`
	QuestionID      *int64           `json:"question_id,omitempty"`
	KnowledgeNodeID *int64           `json:"knowledge_node_id,omitempty"`
	QuestionType    string           `json:"question_type"`
	QuestionContent string           `json:"question_content"`
	Options         json.RawMessage  `json:"options,omitempty"`
	Difficulty      int16            `json:"difficulty"`
	ItemMode        string           `json:"item_mode"`
	SortOrder       int              `json:"sort_order"`
	Status          int16            `json:"status"`
	StudentAnswer   string           `json:"student_answer,omitempty"`
	IsCorrect       *bool            `json:"is_correct,omitempty"`
	Score           *float64         `json:"score,omitempty"`
}

// SubmitAnswerResponse 提交答案响应
type SubmitAnswerResponse struct {
	TaskItemID int64   `json:"task_item_id"`
	IsCorrect  *bool   `json:"is_correct,omitempty"`
	Score      *float64 `json:"score,omitempty"`
	Feedback   string  `json:"feedback,omitempty"`
}

// GenerateTaskResponse 生成任务响应
type GenerateTaskResponse struct {
	TaskID      int64  `json:"task_id"`
	Subject     string `json:"subject"`
	TotalItems  int    `json:"total_items"`
	TaskMode    string `json:"task_mode"`
	Status      string `json:"status"`
}

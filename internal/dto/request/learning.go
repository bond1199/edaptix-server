package request

// LearningUploadRequest 学情素材上传请求
type LearningUploadRequest struct {
	UploadType string `form:"upload_type" binding:"required,oneof=homework exam answer_sheet"`
	Subject    string `form:"subject" binding:"required"`
	Source     string `form:"source" binding:"omitempty,oneof=camera album"` // 默认camera
}

// LearningAnalyzeRequest 触发AI分析请求
type LearningAnalyzeRequest struct {
	UploadID int64 `json:"upload_id" binding:"required"`
}

// ErrorQuestionQueryRequest 错题查询请求
type ErrorQuestionQueryRequest struct {
	Subject  string `form:"subject" binding:"omitempty"`
	Resolved *bool  `form:"resolved" binding:"omitempty"`
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
}

// ErrorQuestionUpdateRequest 修正错题请求
type ErrorQuestionUpdateRequest struct {
	IsResolved    *bool  `json:"is_resolved" binding:"omitempty"`
	KnowledgeNodeID *int64 `json:"knowledge_node_id" binding:"omitempty"`
	ErrorType      string `json:"error_type" binding:"omitempty,oneof=wrong blank guessed"`
}

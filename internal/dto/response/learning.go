package response

// LearningUploadResponse 学情上传响应
type LearningUploadResponse struct {
	UploadID      int64            `json:"upload_id"`
	ValidCount    int              `json:"valid_count"`
	InvalidCount  int              `json:"invalid_count"`
	InvalidReasons []InvalidReason `json:"invalid_reasons,omitempty"`
	Status        string           `json:"status"`
}

// InvalidReason 无效图片原因
type InvalidReason struct {
	PageIndex int    `json:"page_index"`
	Reason    string `json:"reason"`
}

// UploadListResponse 上传历史列表项
type UploadListResponse struct {
	ID         int64  `json:"id"`
	UploadType string `json:"upload_type"`
	Subject    string `json:"subject"`
	Status     int16  `json:"status"`
	PageCount  int    `json:"page_count"`
	CreatedAt  string `json:"created_at"`
}

// UploadDetailResponse 上传详情响应
type UploadDetailResponse struct {
	ID         int64                   `json:"id"`
	UploadType string                  `json:"upload_type"`
	Subject    string                  `json:"subject"`
	Source     string                  `json:"source"`
	Status     int16                   `json:"status"`
	PageCount  int                     `json:"page_count"`
	Items      []UploadItemResponse    `json:"items"`
	CreatedAt  string                  `json:"created_at"`
}

// UploadItemResponse 上传素材项响应
type UploadItemResponse struct {
	ID            int64   `json:"id"`
	ImageURL      string  `json:"image_url"`
	PageIndex     int     `json:"page_index"`
	IsValid       bool    `json:"is_valid"`
	InvalidReason string  `json:"invalid_reason,omitempty"`
}

// ErrorQuestionResponse 错题响应
type ErrorQuestionResponse struct {
	ID               int64   `json:"id"`
	Subject          string  `json:"subject"`
	KnowledgeNodeID  *int64  `json:"knowledge_node_id"`
	QuestionType     string  `json:"question_type"`
	QuestionContent  string  `json:"question_content"`
	CorrectAnswer    string  `json:"correct_answer"`
	StudentAnswer    string  `json:"student_answer"`
	ErrorType        string  `json:"error_type"`
	Difficulty       int16   `json:"difficulty"`
	IsResolved       bool    `json:"is_resolved"`
	ReviewCount      int     `json:"review_count"`
	CreatedAt        string  `json:"created_at"`
}

// LearningStatsResponse 学情统计响应
type LearningStatsResponse struct {
	TotalUploads    int                  `json:"total_uploads"`
	TotalErrors     int                  `json:"total_errors"`
	UnresolvedErrors int                 `json:"unresolved_errors"`
	SubjectStats    []SubjectStatItem    `json:"subject_stats"`
}

// SubjectStatItem 学科统计项
type SubjectStatItem struct {
	Subject         string  `json:"subject"`
	TotalErrors     int     `json:"total_errors"`
	UnresolvedCount int     `json:"unresolved_count"`
	MasteryRate     float64 `json:"mastery_rate"`
}

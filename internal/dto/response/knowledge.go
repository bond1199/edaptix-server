package response

// InitCatalogResponse 从教材目录初始化响应
type InitCatalogResponse struct {
	UploadID  int64  `json:"upload_id"`
	TreeID    int64  `json:"tree_id"`
	Subject   string `json:"subject"`
	NodeCount int    `json:"node_count"`
	Status    string `json:"status"` // processing / completed / failed
}

// KnowledgeTreeResponse 知识树详情响应
type KnowledgeTreeResponse struct {
	ID              int64               `json:"id"`
	Subject         string              `json:"subject"`
	Grade           int16               `json:"grade"`
	TextbookEdition string              `json:"textbook_edition"`
	Status          int16               `json:"status"`
	NodeCount       int                 `json:"node_count"`
	Chapters        []ChapterResponse   `json:"chapters,omitempty"`
}

// ChapterResponse 章节响应
type ChapterResponse struct {
	ID        int64             `json:"id"`
	Name      string            `json:"name"`
	Sections  []SectionResponse `json:"sections,omitempty"`
	SortOrder int               `json:"sort_order"`
}

// SectionResponse 小节响应
type SectionResponse struct {
	ID               int64              `json:"id"`
	Name             string             `json:"name"`
	KnowledgePoints  []KPResponse       `json:"knowledge_points,omitempty"`
	SortOrder        int                `json:"sort_order"`
	MasteryRate      float64            `json:"mastery_rate"`
}

// KPResponse 知识点响应
type KPResponse struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	MasteryRate float64 `json:"mastery_rate"`
	SortOrder   int     `json:"sort_order"`
}

// InitStatusResponse 初始化状态响应
type InitStatusResponse struct {
	Initialized bool                `json:"initialized"`
	Subjects    []SubjectInfoResponse `json:"subjects"`
}

// SubjectInfoResponse 学科信息响应
type SubjectInfoResponse struct {
	Subject         string `json:"subject"`
	Grade           int16  `json:"grade"`
	TextbookEdition string `json:"textbook_edition"`
	NodeCount       int    `json:"node_count"`
}

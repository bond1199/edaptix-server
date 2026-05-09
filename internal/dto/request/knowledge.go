package request

// InitCatalogRequest 从教材目录初始化知识树请求
type InitCatalogRequest struct {
	Subject          string   `json:"subject" binding:"required,oneof=语文 数学 英语 物理 化学 生物 历史 地理 政治"`
	Grade            int      `json:"grade" binding:"required,min=1,max=12"`
	TextbookEdition  string   `json:"textbook_edition"`                              // 教材版本（可选，如人教版、北师大版）
	ImageURLs        []string `json:"image_urls" binding:"required,min=1,max=20"`    // 已上传的图片URL列表
}

// GetInitStatusRequest 获取初始化状态请求（无参数，通过JWT获取userID）

// CompleteInitRequest 完成初始化请求
type CompleteInitRequest struct {
	Subjects []string `json:"subjects"` // 已初始化的学科列表（可选）
}

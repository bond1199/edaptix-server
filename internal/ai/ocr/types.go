package ocr

import "encoding/json"

// OCRMode OCR识别模式
type OCRMode string

const (
	OCRModeCatalog    OCRMode = "catalog"     // 教材目录识别
	OCRModeHomework   OCRMode = "homework"    // 作业识别
	OCRModeHandwriting OCRMode = "handwriting" // 手写答卷识别
)

// OCRResult OCR识别结果
type OCRResult struct {
	Text       string          `json:"text"`
	Layout     []LayoutBlock   `json:"layout,omitempty"`
	Tree       *KnowledgeTree  `json:"tree,omitempty"`
	Engine     string          `json:"engine"`
	Confidence float64         `json:"confidence,omitempty"`
	Words      []OCRWordItem   `json:"words,omitempty"`
	InvalidImages []InvalidImage `json:"invalid_images,omitempty"`
}

// OCRWordItem 单条OCR识别文字
type OCRWordItem struct {
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
	Location   *Location `json:"location,omitempty"`
}

// Location 文字位置信息
type Location struct {
	Left   int `json:"left"`
	Top    int `json:"top"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// LayoutBlock 版面布局块
type LayoutBlock struct {
	Type     string `json:"type"` // title, text, figure, table, formula
	Text     string `json:"text,omitempty"`
	Location *Location `json:"location,omitempty"`
	Children []LayoutBlock `json:"children,omitempty"`
}

// KnowledgeTree AI解析后的知识树结构
type KnowledgeTree struct {
	Subject  string       `json:"subject"`
	Grade    int          `json:"grade"`
	Edition  string       `json:"edition"`
	Chapters []ChapterNode `json:"chapters"`
}

// ChapterNode 章节节点
type ChapterNode struct {
	Name      string        `json:"name"`
	Sections  []SectionNode `json:"sections,omitempty"`
	SortOrder int           `json:"sort_order"`
}

// SectionNode 小节节点
type SectionNode struct {
	Name         string          `json:"name"`
	KnowledgePoints []string    `json:"knowledge_points,omitempty"`
	SortOrder    int             `json:"sort_order"`
}

// InvalidImage 无效图片信息
type InvalidImage struct {
	ImageURL string `json:"image_url"`
	Reason   string `json:"reason"`
}

// BaiduOCRResponse 百度OCR通用响应
type BaiduOCRResponse struct {
	WordsResult []struct {
		Words string `json:"words"`
		Location struct {
			Left   int `json:"left"`
			Top    int `json:"top"`
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"location"`
		Probability struct {
			Average   float64 `json:"average"`
			Min       float64 `json:"min"`
			Variance  float64 `json:"variance"`
		} `json:"probability"`
	} `json:"words_result"`
	WordsResultNum int    `json:"words_result_num"`
	LogID          int64  `json:"log_id"`
	ErrorCode      int    `json:"error_code,omitempty"`
	ErrorMsg       string `json:"error_msg,omitempty"`
}

// BaiduAccessTokenResponse 百度OCR Access Token响应
type BaiduAccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
	SessionKey  string `json:"session_key,omitempty"`
	Error       string `json:"error,omitempty"`
	ErrorDesc   string `json:"error_description,omitempty"`
}

// BaiduTableOCRResponse 百度表格识别响应
type BaiduTableOCRResponse struct {
	TablesResult []struct {
		Body []struct {
			CellLocation []struct {
				Column int `json:"column"`
				Row    int `json:"row"`
			} `json:"cell_location"`
			RowStart int `json:"row_start"`
			RowEnd   int `json:"row_end"`
			Words    string `json:"words"`
		} `json:"body"`
		Header []json.RawMessage `json:"header,omitempty"`
	} `json:"tables_result"`
	TablesResultNum int    `json:"tables_result_num"`
	LogID           int64  `json:"log_id"`
	ErrorCode       int    `json:"error_code,omitempty"`
	ErrorMsg        string `json:"error_msg,omitempty"`
}

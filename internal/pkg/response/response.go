package response

import (
	"time"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

type PageData struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

func Success(c *gin.Context, data interface{}) {
	requestID, _ := c.Get("request_id")
	c.JSON(200, Response{
		Code:      0,
		Message:   "success",
		Data:      data,
		RequestID: toString(requestID),
		Timestamp: time.Now().Unix(),
	})
}

func Error(c *gin.Context, code int, message string) {
	requestID, _ := c.Get("request_id")
	c.JSON(code, Response{
		Code:      code,
		Message:   message,
		RequestID: toString(requestID),
		Timestamp: time.Now().Unix(),
	})
}

func PageSuccess(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	requestID, _ := c.Get("request_id")
	c.JSON(200, Response{
		Code:    0,
		Message: "success",
		Data: PageData{
			List:     list,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
		RequestID: toString(requestID),
		Timestamp: time.Now().Unix(),
	})
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

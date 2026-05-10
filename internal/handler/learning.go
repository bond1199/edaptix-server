package handler

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/edaptix/server/internal/dto/request"
	"github.com/edaptix/server/internal/middleware"
	"github.com/edaptix/server/internal/pkg/response"
	"github.com/edaptix/server/internal/service"
	"github.com/gin-gonic/gin"
)

// LearningHandler 学情数据采集处理器
type LearningHandler struct {
	learningSvc *service.LearningDataService
}

// NewLearningHandler 创建学情采集处理器
func NewLearningHandler(learningSvc *service.LearningDataService) *LearningHandler {
	return &LearningHandler{learningSvc: learningSvc}
}

// Upload 批量上传学情素材
func (h *LearningHandler) Upload(c *gin.Context) {
	userID, exists := c.Get(middleware.ContextUserID)
	if !exists {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req request.LearningUploadRequest
	if err := c.ShouldBind(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// 获取上传的图片
	form, err := c.MultipartForm()
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无法读取上传文件")
		return
	}

	files := form.File["images"]
	if len(files) == 0 {
		response.Error(c, http.StatusBadRequest, "请上传至少一张图片")
		return
	}

	if len(files) > 20 {
		response.Error(c, http.StatusBadRequest, "单次最多上传20张图片")
		return
	}

	// 读取图片数据
	var imageBytesList [][]byte
	var fileNames []string
	for _, file := range files {
		if file.Size > 10*1024*1024 {
			response.Error(c, http.StatusBadRequest, fmt.Sprintf("文件 %s 超过10MB限制", file.Filename))
			return
		}

		f, err := file.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			continue
		}

		imageBytesList = append(imageBytesList, data)
		fileNames = append(fileNames, file.Filename)
	}

	result, err := h.learningSvc.UploadLearningData(
		c.Request.Context(),
		userID.(int64),
		req.UploadType,
		req.Subject,
		req.Source,
		imageBytesList,
		fileNames,
	)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}

// GetUploads 获取上传历史
func (h *LearningHandler) GetUploads(c *gin.Context) {
	userID, exists := c.Get(middleware.ContextUserID)
	if !exists {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	result, err := h.learningSvc.GetUploads(c.Request.Context(), userID.(int64))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}

// GetUploadDetail 获取上传详情
func (h *LearningHandler) GetUploadDetail(c *gin.Context) {
	uploadIDStr := c.Param("id")
	uploadID, err := strconv.ParseInt(uploadIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的上传ID")
		return
	}

	result, err := h.learningSvc.GetUploadDetail(c.Request.Context(), uploadID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}

// GetErrors 获取错题列表
func (h *LearningHandler) GetErrors(c *gin.Context) {
	userID, exists := c.Get(middleware.ContextUserID)
	if !exists {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req request.ErrorQuestionQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	items, total, err := h.learningSvc.GetErrorQuestions(
		c.Request.Context(),
		userID.(int64),
		req.Subject,
		req.Resolved,
		req.Page,
		req.PageSize,
	)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	response.PageSuccess(c, items, total, page, pageSize)
}

// GetStats 获取学情统计
func (h *LearningHandler) GetStats(c *gin.Context) {
	userID, exists := c.Get(middleware.ContextUserID)
	if !exists {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	result, err := h.learningSvc.GetLearningStats(c.Request.Context(), userID.(int64))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}

package handler

import (
	"net/http"
	"strconv"

	"github.com/edaptix/server/internal/dto/request"
	"github.com/edaptix/server/internal/middleware"
	"github.com/edaptix/server/internal/pkg/response"
	"github.com/edaptix/server/internal/service"
	"github.com/gin-gonic/gin"
)

// InitHandler 初始化相关处理器
type InitHandler struct {
	knowledgeTreeSvc *service.KnowledgeTreeService
}

// NewInitHandler 创建初始化处理器
func NewInitHandler(knowledgeTreeSvc *service.KnowledgeTreeService) *InitHandler {
	return &InitHandler{
		knowledgeTreeSvc: knowledgeTreeSvc,
	}
}

// GetInitStatus 获取初始化状态
func (h *InitHandler) GetInitStatus(c *gin.Context) {
	userID, exists := c.Get(middleware.ContextUserID)
	if !exists {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	result, err := h.knowledgeTreeSvc.GetInitStatus(c.Request.Context(), userID.(int64))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}

// InitFromCatalog 从教材目录初始化知识树
func (h *InitHandler) InitFromCatalog(c *gin.Context) {
	userID, exists := c.Get(middleware.ContextUserID)
	if !exists {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req request.InitCatalogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.knowledgeTreeSvc.InitFromCatalog(
		c.Request.Context(),
		userID.(int64),
		req.Subject,
		req.Grade,
		req.TextbookEdition,
		req.ImageURLs,
	)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}

// CompleteInit 完成初始化
func (h *InitHandler) CompleteInit(c *gin.Context) {
	userID, exists := c.Get(middleware.ContextUserID)
	if !exists {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.knowledgeTreeSvc.CompleteInit(c.Request.Context(), userID.(int64)); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "initialization completed"})
}

// GetKnowledgeTree 获取知识树详情
func (h *InitHandler) GetKnowledgeTree(c *gin.Context) {
	treeIDStr := c.Param("id")
	treeID, err := strconv.ParseInt(treeIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid tree id")
		return
	}

	result, err := h.knowledgeTreeSvc.GetKnowledgeTree(c.Request.Context(), treeID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}

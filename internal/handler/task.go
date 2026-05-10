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

// TaskHandler 任务处理器
type TaskHandler struct {
	taskSvc *service.TaskService
}

// NewTaskHandler 创建任务处理器
func NewTaskHandler(taskSvc *service.TaskService) *TaskHandler {
	return &TaskHandler{taskSvc: taskSvc}
}

// GenerateTask 生成每日任务
// POST /api/v1/tasks/generate
func (h *TaskHandler) GenerateTask(c *gin.Context) {
	userID, exists := c.Get(middleware.ContextUserID)
	if !exists {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req request.GenerateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.taskSvc.GenerateDailyTask(
		c.Request.Context(),
		userID.(int64),
		req.Subject,
		req.TaskMode,
	)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}

// GetTodayTask 获取今日任务
// GET /api/v1/tasks/today?subject=数学
func (h *TaskHandler) GetTodayTask(c *gin.Context) {
	userID, exists := c.Get(middleware.ContextUserID)
	if !exists {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	subject := c.Query("subject")
	if subject == "" {
		response.Error(c, http.StatusBadRequest, "缺少subject参数")
		return
	}

	result, err := h.taskSvc.GetTodayTask(c.Request.Context(), userID.(int64), subject)
	if err != nil {
		response.Error(c, http.StatusNotFound, err.Error())
		return
	}

	response.Success(c, result)
}

// GetTaskHistory 获取任务历史
// GET /api/v1/tasks/history
func (h *TaskHandler) GetTaskHistory(c *gin.Context) {
	userID, exists := c.Get(middleware.ContextUserID)
	if !exists {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req request.TaskQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	items, total, err := h.taskSvc.GetTaskHistory(
		c.Request.Context(),
		userID.(int64),
		req.Subject,
		req.Status,
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

// GetTaskDetail 获取任务详情
// GET /api/v1/tasks/:id
func (h *TaskHandler) GetTaskDetail(c *gin.Context) {
	taskIDStr := c.Param("id")
	taskID, err := strconv.ParseInt(taskIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的任务ID")
		return
	}

	result, err := h.taskSvc.GetTaskDetail(c.Request.Context(), taskID)
	if err != nil {
		response.Error(c, http.StatusNotFound, err.Error())
		return
	}

	response.Success(c, result)
}

// StartTask 开始任务
// POST /api/v1/tasks/:id/start
func (h *TaskHandler) StartTask(c *gin.Context) {
	taskIDStr := c.Param("id")
	taskID, err := strconv.ParseInt(taskIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的任务ID")
		return
	}

	if err := h.taskSvc.StartTask(c.Request.Context(), taskID); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, gin.H{"task_id": taskID, "status": "started"})
}

// SubmitAnswer 提交答案
// POST /api/v1/tasks/:id/answer
func (h *TaskHandler) SubmitAnswer(c *gin.Context) {
	taskIDStr := c.Param("id")
	taskID, err := strconv.ParseInt(taskIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的任务ID")
		return
	}

	var req request.SubmitAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// taskItemID 从请求体获取（或从URL路径参数获取）
	// 这里使用 query param item_id 来指定具体的题目
	itemIDStr := c.Query("item_id")
	itemID, err := strconv.ParseInt(itemIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的题目ID")
		return
	}

	_ = taskID // taskID用于权限校验（后续可加）

	result, err := h.taskSvc.SubmitAnswer(c.Request.Context(), itemID, req.Answer, req.AnswerDuration)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, result)
}

// FinishTask 完成任务
// POST /api/v1/tasks/:id/finish
func (h *TaskHandler) FinishTask(c *gin.Context) {
	taskIDStr := c.Param("id")
	taskID, err := strconv.ParseInt(taskIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的任务ID")
		return
	}

	if err := h.taskSvc.FinishTask(c.Request.Context(), taskID); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, gin.H{"task_id": taskID, "status": "finished"})
}

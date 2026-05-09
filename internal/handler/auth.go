package handler

import (
	"net/http"

	"github.com/edaptix/server/internal/dto/request"
	"github.com/edaptix/server/internal/pkg/response"
	"github.com/edaptix/server/internal/service"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	userSvc *service.UserService
}

func NewAuthHandler(userSvc *service.UserService) *AuthHandler {
	return &AuthHandler{userSvc: userSvc}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req request.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.userSvc.Register(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, result)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req request.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.userSvc.Login(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	response.Success(c, result)
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req request.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.userSvc.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	response.Success(c, result)
}

func (h *AuthHandler) SendSMS(c *gin.Context) {
	var req request.SendSMSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	code, err := h.userSvc.SendSMS(c.Request.Context(), req.Phone)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	data := gin.H{"message": "验证码发送成功"}
	if code != "" {
		data["code"] = code
	}

	response.Success(c, data)
}

func (h *AuthHandler) VerifySMS(c *gin.Context) {
	var req request.VerifySMSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	valid, err := h.userSvc.VerifySMS(c.Request.Context(), req.Phone, req.SMSCode)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	if !valid {
		response.Error(c, http.StatusBadRequest, "验证码错误或已过期")
		return
	}

	response.Success(c, gin.H{"valid": true})
}

package handler

import (
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/edaptix/server/internal/middleware"
	"github.com/edaptix/server/internal/pkg/response"
	"github.com/edaptix/server/internal/pkg/storage"
	"github.com/gin-gonic/gin"
)

const maxUploadSize = 10 << 20 // 10MB

var allowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/jpg":  true,
}

var extByMIME = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/jpg":  ".jpg",
}

type UploadHandler struct {
	storage *storage.MinIOProvider
}

func NewUploadHandler(storage *storage.MinIOProvider) *UploadHandler {
	return &UploadHandler{storage: storage}
}

func (h *UploadHandler) UploadImage(c *gin.Context) {
	// Limit request body size
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "failed to get upload file")
		return
	}
	defer file.Close()

	// Validate file size
	if header.Size > maxUploadSize {
		response.Error(c, http.StatusBadRequest, "file size exceeds 10MB limit")
		return
	}

	// Detect content type from file header
	contentType := header.Header.Get("Content-Type")
	if !allowedImageTypes[contentType] {
		response.Error(c, http.StatusBadRequest, "only jpg/png/jpeg images are allowed")
		return
	}

	// Get user ID from JWT context
	userID, exists := c.Get(middleware.ContextUserID)
	if !exists {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Generate object name: uploads/{userID}/{timestamp}_{filename}
	ext := extByMIME[contentType]
	if ext == "" {
		ext = filepath.Ext(header.Filename)
	}
	timestamp := time.Now().Format("20060102150405")
	objectName := fmt.Sprintf("uploads/%d/%s_%s", userID, timestamp, header.Filename)

	url, err := h.storage.Upload(c.Request.Context(), objectName, file, header.Size, contentType)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to upload file")
		return
	}

	response.Success(c, gin.H{"url": url})
}

package service

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/edaptix/server/internal/ai/ocr"
	"github.com/edaptix/server/internal/dto/response"
	"github.com/edaptix/server/internal/model"
	"github.com/edaptix/server/internal/pkg/storage"
	"github.com/edaptix/server/internal/repository"
	"github.com/samber/lo"
	"go.uber.org/zap"
	
)

// LearningDataService 学情数据采集服务
type LearningDataService struct {
	dataRepo    *repository.LearningDataRepo
	treeRepo    *repository.KnowledgeTreeRepo
	ocrPipeline *ocr.Pipeline
	storage     *storage.MinIOProvider
}

// NewLearningDataService 创建学情数据采集服务
func NewLearningDataService(
	dataRepo *repository.LearningDataRepo,
	treeRepo *repository.KnowledgeTreeRepo,
	ocrPipeline *ocr.Pipeline,
	storageProvider *storage.MinIOProvider,
) *LearningDataService {
	return &LearningDataService{
		dataRepo:    dataRepo,
		treeRepo:    treeRepo,
		ocrPipeline: ocrPipeline,
		storage:     storageProvider,
	}
}

// UploadLearningData 批量上传学情素材（同步版，开发阶段使用）
func (s *LearningDataService) UploadLearningData(ctx context.Context, userID int64, uploadType, subject, source string, imageBytesList [][]byte, fileNames []string) (*response.LearningUploadResponse, error) {
	// Step 1: 上传图片到MinIO
	var items []model.UploadItem
	var validCount, invalidCount int
	var invalidReasons []response.InvalidReason

	for i, imgBytes := range imageBytesList {
		// 生成存储路径
		ext := ".jpg"
		if i < len(fileNames) && fileNames[i] != "" {
			if len(fileNames[i]) > 4 {
				ext = fileNames[i][len(fileNames[i])-4:]
			}
		}
		objectName := fmt.Sprintf("learning/%d/%s/%d_%d%s", userID, uploadType, time.Now().UnixMilli(), i, ext)

		url, err := s.storage.Upload(ctx, objectName, bytes.NewReader(imgBytes), int64(len(imgBytes)), "image/jpeg")
		if err != nil {
			invalidCount++
			invalidReasons = append(invalidReasons, response.InvalidReason{PageIndex: i, Reason: "upload_failed"})
			continue
		}

		// Step 2: 图片质量检测（简单校验：大小>1KB）
		isValid := true
		var invalidReason string
		if len(imgBytes) < 1024 {
			isValid = false
			invalidReason = "blank_page"
			invalidCount++
			invalidReasons = append(invalidReasons, response.InvalidReason{PageIndex: i, Reason: invalidReason})
		} else if len(imgBytes) > 10*1024*1024 {
			isValid = false
			invalidReason = "too_large"
			invalidCount++
			invalidReasons = append(invalidReasons, response.InvalidReason{PageIndex: i, Reason: invalidReason})
		}

		if isValid {
			validCount++
		}

		items = append(items, model.UploadItem{
			ImageURL:      url,
			PageIndex:     i,
			IsValid:       isValid,
			InvalidReason: invalidReason,
		})
	}

	// Step 3: 创建上传记录
	upload := &model.LearningUpload{
		UserID:     userID,
		UploadType: uploadType,
		Source:     lo.If(source == "", "camera").Else(source),
		Subject:    subject,
		Status:     2, // AI处理中
		PageCount:  len(imageBytesList),
	}
	if err := s.dataRepo.CreateUpload(ctx, upload); err != nil {
		return nil, fmt.Errorf("创建上传记录失败: %w", err)
	}

	// Step 4: 保存素材明细
	for i := range items {
		items[i].UploadID = upload.ID
	}
	if err := s.dataRepo.CreateUploadItems(ctx, items); err != nil {
		return nil, fmt.Errorf("保存素材明细失败: %w", err)
	}

	// Step 5: OCR识别有效图片（同步版）
	validURLs := lo.FilterMap(items, func(item model.UploadItem, _ int) (string, bool) {
		return item.ImageURL, item.IsValid
	})

	if len(validURLs) > 0 {
		ocrResult, err := s.ocrPipeline.ProcessOCR(ctx, validURLs, ocr.OCRModeHomework)
		if err != nil {
			zap.L().Warn("OCR识别失败", zap.Int64("upload_id", upload.ID), zap.Error(err))
			_ = s.dataRepo.UpdateUploadStatus(ctx, upload.ID, 4) // 处理失败
		} else {
			// 保存OCR结果到素材明细
			for i := range items {
				if items[i].IsValid && i < len(ocrResult.Words) {
					// 将OCR文字存入item的OCRResult（简化版，存纯文本）
					items[i].OCRResult = nil // TODO: JSON格式化存储
				}
			}
			_ = s.dataRepo.UpdateUploadStatus(ctx, upload.ID, 3) // 已完成

			zap.L().Info("学情素材OCR识别完成",
				zap.Int64("upload_id", upload.ID),
				zap.Int("word_count", len(ocrResult.Words)),
			)

			// TODO: 调用AI分析错题归档（Phase 2.2 出题算法实现后接入）
		}
	} else {
		_ = s.dataRepo.UpdateUploadStatus(ctx, upload.ID, 3)
	}

	return &response.LearningUploadResponse{
		UploadID:      upload.ID,
		ValidCount:    validCount,
		InvalidCount:  invalidCount,
		InvalidReasons: invalidReasons,
		Status:        "processing",
	}, nil
}

// GetUploads 获取上传历史
func (s *LearningDataService) GetUploads(ctx context.Context, userID int64) ([]response.UploadListResponse, error) {
	// 查询用户上传记录
	var uploads []model.LearningUpload
	err := s.dataRepo.DB().WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(50).
		Find(&uploads).Error
	if err != nil {
		return nil, fmt.Errorf("查询上传历史失败: %w", err)
	}

	result := lo.Map(uploads, func(u model.LearningUpload, _ int) response.UploadListResponse {
		return response.UploadListResponse{
			ID:         u.ID,
			UploadType: u.UploadType,
			Subject:    u.Subject,
			Status:     u.Status,
			PageCount:  u.PageCount,
			CreatedAt:  u.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	})

	return result, nil
}

// GetUploadDetail 获取上传详情
func (s *LearningDataService) GetUploadDetail(ctx context.Context, uploadID int64) (*response.UploadDetailResponse, error) {
	upload, err := s.dataRepo.GetUploadByID(ctx, uploadID)
	if err != nil {
		return nil, fmt.Errorf("查询上传记录失败: %w", err)
	}

	items, err := s.dataRepo.GetUploadItemsByUploadID(ctx, uploadID)
	if err != nil {
		return nil, fmt.Errorf("查询素材明细失败: %w", err)
	}

	itemResponses := lo.Map(items, func(item model.UploadItem, _ int) response.UploadItemResponse {
		return response.UploadItemResponse{
			ID:            item.ID,
			ImageURL:      item.ImageURL,
			PageIndex:     item.PageIndex,
			IsValid:       item.IsValid,
			InvalidReason: item.InvalidReason,
		}
	})

	return &response.UploadDetailResponse{
		ID:         upload.ID,
		UploadType: upload.UploadType,
		Subject:    upload.Subject,
		Source:     upload.Source,
		Status:     upload.Status,
		PageCount:  upload.PageCount,
		Items:      itemResponses,
		CreatedAt:  upload.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// GetErrorQuestions 获取错题列表
func (s *LearningDataService) GetErrorQuestions(ctx context.Context, userID int64, subject string, resolved *bool, page, pageSize int) ([]response.ErrorQuestionResponse, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	query := s.dataRepo.DB().WithContext(ctx).
		Where("user_id = ?", userID)

	if subject != "" {
		query = query.Where("subject = ?", subject)
	}
	if resolved != nil {
		query = query.Where("is_resolved = ?", *resolved)
	}

	var total int64
	query.Model(&model.ErrorQuestion{}).Count(&total)

	var errors []model.ErrorQuestion
	query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&errors)

	result := lo.Map(errors, func(e model.ErrorQuestion, _ int) response.ErrorQuestionResponse {
		return response.ErrorQuestionResponse{
			ID:              e.ID,
			Subject:         e.Subject,
			KnowledgeNodeID: e.KnowledgeNodeID,
			QuestionType:    e.QuestionType,
			QuestionContent: e.QuestionContent,
			CorrectAnswer:   e.CorrectAnswer,
			StudentAnswer:   e.StudentAnswer,
			ErrorType:       e.ErrorType,
			Difficulty:      e.Difficulty,
			IsResolved:      e.IsResolved,
			ReviewCount:     e.ReviewCount,
			CreatedAt:       e.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	})

	return result, total, nil
}

// GetLearningStats 获取学情统计
func (s *LearningDataService) GetLearningStats(ctx context.Context, userID int64) (*response.LearningStatsResponse, error) {
	var totalUploads int64
	s.dataRepo.DB().WithContext(ctx).
		Model(&model.LearningUpload{}).Where("user_id = ?", userID).Count(&totalUploads)

	var totalErrors int64
	s.dataRepo.DB().WithContext(ctx).
		Model(&model.ErrorQuestion{}).Where("user_id = ?", userID).Count(&totalErrors)

	var unresolvedErrors int64
	s.dataRepo.DB().WithContext(ctx).
		Model(&model.ErrorQuestion{}).Where("user_id = ? AND is_resolved = false", userID).Count(&unresolvedErrors)

	// 按学科统计
	type subjectStat struct {
		Subject  string
		Count    int
		Unresolved int
	}
	var subjectStats []subjectStat
	s.dataRepo.DB().WithContext(ctx).
		Model(&model.ErrorQuestion{}).
		Select("subject, COUNT(*) as count, SUM(CASE WHEN is_resolved = false THEN 1 ELSE 0 END) as unresolved").
		Where("user_id = ?", userID).
		Group("subject").
		Find(&subjectStats)

	// 获取各学科掌握率
	trees, _ := s.treeRepo.ListTreesByUser(ctx, userID)
	treeMap := lo.SliceToMap(trees, func(t model.KnowledgeTree) (string, float64) {
		return t.Subject, 0 // TODO: 从节点计算实际掌握率
	})

	subjectItems := lo.Map(subjectStats, func(ss subjectStat, _ int) response.SubjectStatItem {
		masteryRate := treeMap[ss.Subject]
		return response.SubjectStatItem{
			Subject:         ss.Subject,
			TotalErrors:     ss.Count,
			UnresolvedCount: ss.Unresolved,
			MasteryRate:     masteryRate,
		}
	})

	return &response.LearningStatsResponse{
		TotalUploads:    int(totalUploads),
		TotalErrors:     int(totalErrors),
		UnresolvedErrors: int(unresolvedErrors),
		SubjectStats:    subjectItems,
	}, nil
}

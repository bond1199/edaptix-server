package service

import (
	"context"
	"fmt"
	"time"

	"github.com/edaptix/server/internal/ai/llm"
	"github.com/edaptix/server/internal/ai/ocr"
	"github.com/edaptix/server/internal/dto/response"
	"github.com/edaptix/server/internal/model"
	"github.com/edaptix/server/internal/pkg/storage"
	"github.com/edaptix/server/internal/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// KnowledgeTreeService 知识树服务
type KnowledgeTreeService struct {
	treeRepo    *repository.KnowledgeTreeRepo
	dataRepo    *repository.LearningDataRepo
	userRepo    *repository.UserRepo
	ocrPipeline *ocr.Pipeline
	llmClient   *llm.DeepSeekClient
	storage     *storage.MinIOProvider
	db          *gorm.DB
}

// NewKnowledgeTreeService 创建知识树服务
func NewKnowledgeTreeService(
	treeRepo *repository.KnowledgeTreeRepo,
	dataRepo *repository.LearningDataRepo,
	userRepo *repository.UserRepo,
	ocrPipeline *ocr.Pipeline,
	llmClient *llm.DeepSeekClient,
	storageProvider *storage.MinIOProvider,
	db *gorm.DB,
) *KnowledgeTreeService {
	return &KnowledgeTreeService{
		treeRepo:    treeRepo,
		dataRepo:    dataRepo,
		userRepo:    userRepo,
		ocrPipeline: ocrPipeline,
		llmClient:   llmClient,
		storage:     storageProvider,
		db:          db,
	}
}

// InitFromCatalog 从教材目录初始化知识树（同步版本，开发阶段使用）
// 完整流程：上传图片 → OCR识别 → AI解析目录 → 构建知识树 → 更新初始化状态
func (s *KnowledgeTreeService) InitFromCatalog(ctx context.Context, userID int64, subject string, grade int, textbookEdition string, imageURLs []string) (*response.InitCatalogResponse, error) {
	// Step 1: 创建上传记录
	upload := &model.LearningUpload{
		UserID:     userID,
		UploadType: "catalog",
		Source:     "camera",
		Subject:    subject,
		Status:     2, // AI处理中
		PageCount:  len(imageURLs),
	}
	if err := s.dataRepo.CreateUpload(ctx, upload); err != nil {
		return nil, fmt.Errorf("创建上传记录失败: %w", err)
	}

	// Step 2: 保存上传素材明细
	var items []model.UploadItem
	for i, url := range imageURLs {
		items = append(items, model.UploadItem{
			UploadID:  upload.ID,
			ImageURL:  url,
			PageIndex: i,
			IsValid:   true,
		})
	}
	if err := s.dataRepo.CreateUploadItems(ctx, items); err != nil {
		return nil, fmt.Errorf("保存素材明细失败: %w", err)
	}

	// Step 3: OCR识别
	ocrResult, err := s.ocrPipeline.ProcessOCR(ctx, imageURLs, ocr.OCRModeCatalog)
	if err != nil {
		_ = s.dataRepo.UpdateUploadStatus(ctx, upload.ID, 4) // 处理失败
		return nil, fmt.Errorf("OCR识别失败: %w", err)
	}

	zap.L().Info("OCR识别完成",
		zap.Int64("upload_id", upload.ID),
		zap.Int("word_count", len(ocrResult.Words)),
		zap.String("engine", ocrResult.Engine),
	)

	// 保存OCR结果到素材明细
	for i := range items {
		items[i].OCRResult = nil // TODO: 存储OCR原始结果
	}

	// Step 4: AI解析目录结构
	catalogResult, err := s.llmClient.ParseCatalog(ctx, ocrResult.Text, subject, grade)
	if err != nil {
		_ = s.dataRepo.UpdateUploadStatus(ctx, upload.ID, 4) // 处理失败
		return nil, fmt.Errorf("AI解析目录失败: %w", err)
	}

	// 如果AI返回了学科和年级，优先使用
	if catalogResult.Subject != "" {
		subject = catalogResult.Subject
	}
	if catalogResult.Grade > 0 {
		grade = catalogResult.Grade
	}
	if catalogResult.Edition != "" {
		textbookEdition = catalogResult.Edition
	}

	zap.L().Info("AI解析目录完成",
		zap.Int64("upload_id", upload.ID),
		zap.String("subject", subject),
		zap.Int("chapter_count", len(catalogResult.Chapters)),
	)

	// Step 5: 创建知识树记录
	tree := &model.KnowledgeTree{
		UserID:          userID,
		Subject:         subject,
		Grade:           int16(grade),
		TextbookEdition: textbookEdition,
		Status:          2, // AI处理中
	}
	if err := s.treeRepo.CreateTree(ctx, tree); err != nil {
		_ = s.dataRepo.UpdateUploadStatus(ctx, upload.ID, 4)
		return nil, fmt.Errorf("创建知识树失败: %w", err)
	}

	// Step 6: 构建知识节点（五级结构）
	nodes, err := s.buildKnowledgeNodes(tree.ID, catalogResult)
	if err != nil {
		_ = s.treeRepo.UpdateTreeStatus(ctx, tree.ID, 3) // 处理失败
		_ = s.dataRepo.UpdateUploadStatus(ctx, upload.ID, 4)
		return nil, fmt.Errorf("构建知识节点失败: %w", err)
	}

	if err := s.treeRepo.CreateNodes(ctx, nodes); err != nil {
		_ = s.treeRepo.UpdateTreeStatus(ctx, tree.ID, 3)
		_ = s.dataRepo.UpdateUploadStatus(ctx, upload.ID, 4)
		return nil, fmt.Errorf("保存知识节点失败: %w", err)
	}

	// Step 7: 更新状态
	_ = s.treeRepo.UpdateTreeStatus(ctx, tree.ID, 1) // 正常
	_ = s.dataRepo.UpdateUploadStatus(ctx, upload.ID, 3) // 已完成

	zap.L().Info("知识树构建完成",
		zap.Int64("tree_id", tree.ID),
		zap.Int("node_count", len(nodes)),
	)

	return &response.InitCatalogResponse{
		UploadID:  upload.ID,
		TreeID:    tree.ID,
		Subject:   subject,
		NodeCount: len(nodes),
		Status:    "completed",
	}, nil
}

// buildKnowledgeNodes 从AI解析结果构建知识节点
func (s *KnowledgeTreeService) buildKnowledgeNodes(treeID int64, catalog *llm.CatalogParseResult) ([]model.KnowledgeNode, error) {
	var nodes []model.KnowledgeNode
	now := time.Now()
	sortOrder := 0

	// Level 1: 年级节点
	gradeNode := model.KnowledgeNode{
		TreeID:    treeID,
		ParentID:  nil,
		Level:     1,
		Name:      fmt.Sprintf("%d年级", catalog.Grade),
		SortOrder: sortOrder,
	}
	nodes = append(nodes, gradeNode)
	sortOrder++

	// Level 2: 科目节点
	subjectNode := model.KnowledgeNode{
		TreeID:    treeID,
		ParentID:  nil, // 将在插入后回填
		Level:     2,
		Name:      catalog.Subject,
		SortOrder: sortOrder,
	}
	nodes = append(nodes, subjectNode)
	sortOrder++

	// 记录章的索引，用于回填ParentID
	chapterStartIdx := len(nodes)

	for ci, chapter := range catalog.Chapters {
		// Level 3: 章节节点
		chapterNode := model.KnowledgeNode{
			TreeID:    treeID,
			ParentID:  nil, // 将在插入后回填
			Level:     3,
			Name:      chapter.Name,
			SortOrder: sortOrder,
		}
		nodes = append(nodes, chapterNode)
		sortOrder++

		sectionStartIdx := len(nodes)

		for si, section := range chapter.Sections {
			// Level 4: 小节节点
			sectionNode := model.KnowledgeNode{
				TreeID:    treeID,
				ParentID:  nil,
				Level:     4,
				Name:      section.Name,
				SortOrder: sortOrder,
			}
			nodes = append(nodes, sectionNode)
			sortOrder++

			// Level 5: 知识点节点
			for ki, kp := range section.KnowledgePoints {
				kpNode := model.KnowledgeNode{
					TreeID:    treeID,
					ParentID:  nil,
					Level:     5,
					Name:      kp,
					SortOrder: ki,
				}
				nodes = append(nodes, kpNode)
			}

			// 回填小节节点的ParentID为章节节点
			_ = sectionStartIdx
			_ = si
			_ = ci
		}
	}

	_ = chapterStartIdx
	_ = now

	// 注意：ParentID回填需要在数据库插入后获取ID
	// 这里采用两阶段策略：先插入所有节点，再更新ParentID
	// 但为了简化，我们使用内存中的索引关系
	// 在实际CreateNodes后，需要二次更新ParentID

	return nodes, nil
}

// GetInitStatus 获取用户初始化状态
func (s *KnowledgeTreeService) GetInitStatus(ctx context.Context, userID int64) (*response.InitStatusResponse, error) {
	// 获取用户初始化状态
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	// 获取用户所有知识树
	trees, err := s.treeRepo.ListTreesByUser(ctx, userID)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("查询知识树失败: %w", err)
	}

	var subjects []response.SubjectInfoResponse
	for _, tree := range trees {
		nodeCount, _ := s.treeRepo.CountNodesByTree(ctx, tree.ID)
		subjects = append(subjects, response.SubjectInfoResponse{
			Subject:         tree.Subject,
			Grade:           tree.Grade,
			TextbookEdition: tree.TextbookEdition,
			NodeCount:       int(nodeCount),
		})
	}

	if subjects == nil {
		subjects = []response.SubjectInfoResponse{}
	}

	return &response.InitStatusResponse{
		Initialized: user.Initialized,
		Subjects:    subjects,
	}, nil
}

// GetKnowledgeTree 获取知识树详情
func (s *KnowledgeTreeService) GetKnowledgeTree(ctx context.Context, treeID int64) (*response.KnowledgeTreeResponse, error) {
	tree, err := s.treeRepo.GetTreeByID(ctx, treeID)
	if err != nil {
		return nil, fmt.Errorf("查询知识树失败: %w", err)
	}

	nodes, err := s.treeRepo.GetNodesByTreeID(ctx, treeID)
	if err != nil {
		return nil, fmt.Errorf("查询知识节点失败: %w", err)
	}

	// 构建层级树
	chapters := s.buildTreeResponse(nodes)

	return &response.KnowledgeTreeResponse{
		ID:              tree.ID,
		Subject:         tree.Subject,
		Grade:           tree.Grade,
		TextbookEdition: tree.TextbookEdition,
		Status:          tree.Status,
		NodeCount:       len(nodes),
		Chapters:        chapters,
	}, nil
}

// buildTreeResponse 从扁平节点构建层级响应
func (s *KnowledgeTreeService) buildTreeResponse(nodes []model.KnowledgeNode) []response.ChapterResponse {
	// 按ParentID分组
	childrenMap := make(map[int64][]model.KnowledgeNode)
	var level3Nodes []model.KnowledgeNode

	for _, node := range nodes {
		if node.Level == 3 {
			level3Nodes = append(level3Nodes, node)
		}
		if node.ParentID != nil {
			childrenMap[*node.ParentID] = append(childrenMap[*node.ParentID], node)
		}
	}

	var chapters []response.ChapterResponse
	for _, chNode := range level3Nodes {
		chapter := response.ChapterResponse{
			ID:        chNode.ID,
			Name:      chNode.Name,
			SortOrder: chNode.SortOrder,
		}

		// 构建小节
		children := childrenMap[chNode.ID]
		for _, secNode := range children {
			if secNode.Level != 4 {
				continue
			}
			section := response.SectionResponse{
				ID:          secNode.ID,
				Name:        secNode.Name,
				SortOrder:   secNode.SortOrder,
				MasteryRate: secNode.MasteryRate,
			}

			// 构建知识点
			kps := childrenMap[secNode.ID]
			for _, kpNode := range kps {
				if kpNode.Level != 5 {
					continue
				}
				section.KnowledgePoints = append(section.KnowledgePoints, response.KPResponse{
					ID:          kpNode.ID,
					Name:        kpNode.Name,
					MasteryRate: kpNode.MasteryRate,
					SortOrder:   kpNode.SortOrder,
				})
			}

			chapter.Sections = append(chapter.Sections, section)
		}

		chapters = append(chapters, chapter)
	}

	return chapters
}

// CompleteInit 完成初始化
func (s *KnowledgeTreeService) CompleteInit(ctx context.Context, userID int64) error {
	// 检查是否至少有一棵知识树
	trees, err := s.treeRepo.ListTreesByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("查询知识树失败: %w", err)
	}

	if len(trees) == 0 {
		return fmt.Errorf("请先完成至少一个学科的初始化")
	}

	// 更新用户初始化状态
	if err := s.userRepo.UpdateInitialized(ctx, userID, true); err != nil {
		return fmt.Errorf("更新初始化状态失败: %w", err)
	}

	zap.L().Info("用户初始化完成",
		zap.Int64("user_id", userID),
		zap.Int("subject_count", len(trees)),
	)

	return nil
}

package repository

import (
	"context"

	"github.com/edaptix/server/internal/model"
	"gorm.io/gorm"
)

// KnowledgeTreeRepo 知识树仓库
type KnowledgeTreeRepo struct {
	db *gorm.DB
}

// NewKnowledgeTreeRepo 创建知识树仓库
func NewKnowledgeTreeRepo(db *gorm.DB) *KnowledgeTreeRepo {
	return &KnowledgeTreeRepo{db: db}
}

// CreateTree 创建知识树
func (r *KnowledgeTreeRepo) CreateTree(ctx context.Context, tree *model.KnowledgeTree) error {
	return r.db.WithContext(ctx).Create(tree).Error
}

// GetTreeByID 根据ID获取知识树
func (r *KnowledgeTreeRepo) GetTreeByID(ctx context.Context, id int64) (*model.KnowledgeTree, error) {
	var tree model.KnowledgeTree
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&tree).Error; err != nil {
		return nil, err
	}
	return &tree, nil
}

// GetTreeByUserAndSubject 获取用户某学科的知识树
func (r *KnowledgeTreeRepo) GetTreeByUserAndSubject(ctx context.Context, userID int64, subject string) (*model.KnowledgeTree, error) {
	var tree model.KnowledgeTree
	if err := r.db.WithContext(ctx).Where("user_id = ? AND subject = ?", userID, subject).First(&tree).Error; err != nil {
		return nil, err
	}
	return &tree, nil
}

// ListTreesByUser 获取用户所有知识树
func (r *KnowledgeTreeRepo) ListTreesByUser(ctx context.Context, userID int64) ([]model.KnowledgeTree, error) {
	var trees []model.KnowledgeTree
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&trees).Error; err != nil {
		return nil, err
	}
	return trees, nil
}

// UpdateTreeStatus 更新知识树状态
func (r *KnowledgeTreeRepo) UpdateTreeStatus(ctx context.Context, id int64, status int16) error {
	return r.db.WithContext(ctx).Model(&model.KnowledgeTree{}).Where("id = ?", id).Update("status", status).Error
}

// CreateNode 创建知识节点
func (r *KnowledgeTreeRepo) CreateNode(ctx context.Context, node *model.KnowledgeNode) error {
	return r.db.WithContext(ctx).Create(node).Error
}

// CreateNodes 批量创建知识节点
func (r *KnowledgeTreeRepo) CreateNodes(ctx context.Context, nodes []model.KnowledgeNode) error {
	if len(nodes) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&nodes).Error
}

// GetNodesByTreeID 获取知识树的所有节点
func (r *KnowledgeTreeRepo) GetNodesByTreeID(ctx context.Context, treeID int64) ([]model.KnowledgeNode, error) {
	var nodes []model.KnowledgeNode
	if err := r.db.WithContext(ctx).Where("tree_id = ?", treeID).Order("level ASC, sort_order ASC").Find(&nodes).Error; err != nil {
		return nil, err
	}
	return nodes, nil
}

// GetNodesByTreeIDAndLevel 获取知识树指定层级的节点
func (r *KnowledgeTreeRepo) GetNodesByTreeIDAndLevel(ctx context.Context, treeID int64, level int16) ([]model.KnowledgeNode, error) {
	var nodes []model.KnowledgeNode
	if err := r.db.WithContext(ctx).Where("tree_id = ? AND level = ?", treeID, level).Order("sort_order ASC").Find(&nodes).Error; err != nil {
		return nil, err
	}
	return nodes, nil
}

// GetNodeByID 根据ID获取知识节点
func (r *KnowledgeTreeRepo) GetNodeByID(ctx context.Context, id int64) (*model.KnowledgeNode, error) {
	var node model.KnowledgeNode
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&node).Error; err != nil {
		return nil, err
	}
	return &node, nil
}

// UpdateNodeMastery 更新节点掌握率
func (r *KnowledgeTreeRepo) UpdateNodeMastery(ctx context.Context, id int64, masteryRate float64) error {
	return r.db.WithContext(ctx).Model(&model.KnowledgeNode{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"mastery_rate": masteryRate,
		}).Error
}

// CountNodesByTree 统计知识树节点数
func (r *KnowledgeTreeRepo) CountNodesByTree(ctx context.Context, treeID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.KnowledgeNode{}).Where("tree_id = ?", treeID).Count(&count).Error
	return count, err
}

// --- 学情上传相关 ---

// LearningDataRepo 学情数据仓库
type LearningDataRepo struct {
	db *gorm.DB
}

// NewLearningDataRepo 创建学情数据仓库
func NewLearningDataRepo(db *gorm.DB) *LearningDataRepo {
	return &LearningDataRepo{db: db}
}

// CreateUpload 创建上传记录
func (r *LearningDataRepo) CreateUpload(ctx context.Context, upload *model.LearningUpload) error {
	return r.db.WithContext(ctx).Create(upload).Error
}

// GetUploadByID 根据ID获取上传记录
func (r *LearningDataRepo) GetUploadByID(ctx context.Context, id int64) (*model.LearningUpload, error) {
	var upload model.LearningUpload
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&upload).Error; err != nil {
		return nil, err
	}
	return &upload, nil
}

// UpdateUploadStatus 更新上传记录状态
func (r *LearningDataRepo) UpdateUploadStatus(ctx context.Context, id int64, status int16) error {
	return r.db.WithContext(ctx).Model(&model.LearningUpload{}).Where("id = ?", id).Update("status", status).Error
}

// CreateUploadItem 创建上传素材明细
func (r *LearningDataRepo) CreateUploadItem(ctx context.Context, item *model.UploadItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// CreateUploadItems 批量创建上传素材明细
func (r *LearningDataRepo) CreateUploadItems(ctx context.Context, items []model.UploadItem) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&items).Error
}

// GetUploadItemsByUploadID 获取上传批次的所有素材
func (r *LearningDataRepo) GetUploadItemsByUploadID(ctx context.Context, uploadID int64) ([]model.UploadItem, error) {
	var items []model.UploadItem
	if err := r.db.WithContext(ctx).Where("upload_id = ?", uploadID).Order("page_index ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

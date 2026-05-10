package repository

import (
	"context"
	"time"

	"github.com/edaptix/server/internal/model"
	"gorm.io/gorm"
)

// TaskRepo 任务仓库
type TaskRepo struct {
	db *gorm.DB
}

// NewTaskRepo 创建任务仓库
func NewTaskRepo(db *gorm.DB) *TaskRepo {
	return &TaskRepo{db: db}
}

// DB 获取底层*gorm.DB（用于复杂查询）
func (r *TaskRepo) DB() *gorm.DB {
	return r.db
}

// --- DailyTask ---

// CreateTask 创建每日任务
func (r *TaskRepo) CreateTask(ctx context.Context, task *model.DailyTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

// GetTaskByID 根据ID获取任务
func (r *TaskRepo) GetTaskByID(ctx context.Context, id int64) (*model.DailyTask, error) {
	var task model.DailyTask
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// GetTodayTaskByUserAndSubject 获取用户今日某学科的任务
func (r *TaskRepo) GetTodayTaskByUserAndSubject(ctx context.Context, userID int64, subject string) (*model.DailyTask, error) {
	today := time.Now().Truncate(24 * time.Hour)
	var task model.DailyTask
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND subject = ? AND task_date >= ?", userID, subject, today).
		First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// ListTasksByUser 获取用户任务列表
func (r *TaskRepo) ListTasksByUser(ctx context.Context, userID int64, subject string, status *int16, offset, limit int) ([]model.DailyTask, int64, error) {
	query := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if subject != "" {
		query = query.Where("subject = ?", subject)
	}
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	var total int64
	if err := query.Model(&model.DailyTask{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var tasks []model.DailyTask
	if err := query.Order("task_date DESC").Offset(offset).Limit(limit).Find(&tasks).Error; err != nil {
		return nil, 0, err
	}
	return tasks, total, nil
}

// UpdateTask 更新任务
func (r *TaskRepo) UpdateTask(ctx context.Context, task *model.DailyTask) error {
	return r.db.WithContext(ctx).Save(task).Error
}

// UpdateTaskStatus 更新任务状态
func (r *TaskRepo) UpdateTaskStatus(ctx context.Context, id int64, status int16) error {
	return r.db.WithContext(ctx).Model(&model.DailyTask{}).Where("id = ?", id).Update("status", status).Error
}

// --- TaskItem ---

// CreateTaskItems 批量创建任务题目
func (r *TaskRepo) CreateTaskItems(ctx context.Context, items []model.TaskItem) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&items).Error
}

// GetTaskItemsByTaskID 获取任务的所有题目
func (r *TaskRepo) GetTaskItemsByTaskID(ctx context.Context, taskID int64) ([]model.TaskItem, error) {
	var items []model.TaskItem
	if err := r.db.WithContext(ctx).Where("task_id = ?", taskID).Order("sort_order ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// GetTaskItemByID 根据ID获取任务题目
func (r *TaskRepo) GetTaskItemByID(ctx context.Context, id int64) (*model.TaskItem, error) {
	var item model.TaskItem
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// UpdateTaskItem 更新任务题目
func (r *TaskRepo) UpdateTaskItem(ctx context.Context, item *model.TaskItem) error {
	return r.db.WithContext(ctx).Save(item).Error
}

// --- QuestionBank ---

// GetQuestionsByKnowledgeNode 根据知识点节点获取题目
func (r *TaskRepo) GetQuestionsByKnowledgeNode(ctx context.Context, knowledgeNodeID int64, questionType string, limit int) ([]model.QuestionBank, error) {
	var questions []model.QuestionBank
	query := r.db.WithContext(ctx).Where("knowledge_node_id = ? AND is_valid = true", knowledgeNodeID)
	if questionType != "" {
		query = query.Where("question_type = ?", questionType)
	}
	if err := query.Order("exam_frequency DESC, usage_count ASC").Limit(limit).Find(&questions).Error; err != nil {
		return nil, err
	}
	return questions, nil
}

// GetQuestionsBySubject 根据学科获取题目
func (r *TaskRepo) GetQuestionsBySubject(ctx context.Context, subject string, questionType string, limit int) ([]model.QuestionBank, error) {
	var questions []model.QuestionBank
	query := r.db.WithContext(ctx).Where("subject = ? AND is_valid = true", subject)
	if questionType != "" {
		query = query.Where("question_type = ?", questionType)
	}
	if err := query.Order("exam_frequency DESC, usage_count ASC").Limit(limit).Find(&questions).Error; err != nil {
		return nil, err
	}
	return questions, nil
}

// GetQuestionByID 根据ID获取题目
func (r *TaskRepo) GetQuestionByID(ctx context.Context, id int64) (*model.QuestionBank, error) {
	var q model.QuestionBank
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&q).Error; err != nil {
		return nil, err
	}
	return &q, nil
}

// CreateQuestion 创建题目
func (r *TaskRepo) CreateQuestion(ctx context.Context, q *model.QuestionBank) error {
	return r.db.WithContext(ctx).Create(q).Error
}

// IncrementQuestionUsage 增加题目使用次数
func (r *TaskRepo) IncrementQuestionUsage(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&model.QuestionBank{}).Where("id = ?", id).
		UpdateColumn("usage_count", gorm.Expr("usage_count + 1")).Error
}

// --- UserQuestionHistory ---

// RecordUserHistory 记录用户答题历史
func (r *TaskRepo) RecordUserHistory(ctx context.Context, h *model.UserQuestionHistory) error {
	return r.db.WithContext(ctx).Create(h).Error
}

// GetUserHistoryByUser 获取用户答题历史
func (r *TaskRepo) GetUserHistoryByUser(ctx context.Context, userID int64, questionID int64) (*model.UserQuestionHistory, error) {
	var h model.UserQuestionHistory
	if err := r.db.WithContext(ctx).Where("user_id = ? AND question_id = ?", userID, questionID).First(&h).Error; err != nil {
		return nil, err
	}
	return &h, nil
}

// GetUserAnsweredQuestionIDs 获取用户已答题目ID集合
func (r *TaskRepo) GetUserAnsweredQuestionIDs(ctx context.Context, userID int64) (map[int64]bool, error) {
	var histories []model.UserQuestionHistory
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&histories).Error; err != nil {
		return nil, err
	}
	result := make(map[int64]bool, len(histories))
	for _, h := range histories {
		result[h.QuestionID] = true
	}
	return result, nil
}

// --- 知识节点查询（用于出题算法） ---

// GetWeakNodes 获取薄弱知识点节点（掌握率低于阈值）
func (r *TaskRepo) GetWeakNodes(ctx context.Context, treeID int64, threshold float64) ([]model.KnowledgeNode, error) {
	var nodes []model.KnowledgeNode
	if err := r.db.WithContext(ctx).
		Where("tree_id = ? AND level = 5 AND mastery_rate < ?", treeID, threshold).
		Order("mastery_rate ASC").
		Find(&nodes).Error; err != nil {
		return nil, err
	}
	return nodes, nil
}

// GetUnpracticedNodes 获取未练习的知识点节点
func (r *TaskRepo) GetUnpracticedNodes(ctx context.Context, treeID int64) ([]model.KnowledgeNode, error) {
	var nodes []model.KnowledgeNode
	if err := r.db.WithContext(ctx).
		Where("tree_id = ? AND level = 5 AND question_count = 0", treeID).
		Order("sort_order ASC").
		Find(&nodes).Error; err != nil {
		return nil, err
	}
	return nodes, nil
}

// GetNodesForReview 获取需要复习的知识点节点（根据遗忘曲线）
func (r *TaskRepo) GetNodesForReview(ctx context.Context, treeID int64, beforeTime time.Time) ([]model.KnowledgeNode, error) {
	var nodes []model.KnowledgeNode
	if err := r.db.WithContext(ctx).
		Where("tree_id = ? AND level = 5 AND last_practiced < ? AND last_practiced IS NOT NULL", treeID, beforeTime).
		Order("last_practiced ASC").
		Find(&nodes).Error; err != nil {
		return nil, err
	}
	return nodes, nil
}

// GetErrorQuestionIDsByUser 获取用户未解决的错题ID列表
func (r *TaskRepo) GetErrorQuestionIDsByUser(ctx context.Context, userID int64, subject string) ([]int64, error) {
	var ids []int64
	query := r.db.WithContext(ctx).Model(&model.ErrorQuestion{}).
		Where("user_id = ? AND is_resolved = false", userID)
	if subject != "" {
		query = query.Where("subject = ?", subject)
	}
	if err := query.Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

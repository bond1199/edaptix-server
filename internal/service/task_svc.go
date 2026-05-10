package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/edaptix/server/internal/ai/llm"
	"github.com/edaptix/server/internal/dto/response"
	"github.com/edaptix/server/internal/model"
	"github.com/edaptix/server/internal/repository"
	"go.uber.org/zap"
)

// TaskService 每日任务服务
type TaskService struct {
	taskRepo     *repository.TaskRepo
	treeRepo     *repository.KnowledgeTreeRepo
	userRepo     *repository.UserRepo
	algorithm    *QuestionAlgorithm
	llmClient    *llm.DeepSeekClient
}

// NewTaskService 创建任务服务
func NewTaskService(
	taskRepo *repository.TaskRepo,
	treeRepo *repository.KnowledgeTreeRepo,
	userRepo *repository.UserRepo,
	algorithm *QuestionAlgorithm,
	llmClient *llm.DeepSeekClient,
) *TaskService {
	return &TaskService{
		taskRepo:  taskRepo,
		treeRepo:  treeRepo,
		userRepo:  userRepo,
		algorithm: algorithm,
		llmClient: llmClient,
	}
}

// GenerateDailyTask 生成每日任务
func (s *TaskService) GenerateDailyTask(ctx context.Context, userID int64, subject, taskMode string) (*response.GenerateTaskResponse, error) {
	// 检查今日是否已生成该学科任务
	existing, err := s.taskRepo.GetTodayTaskByUserAndSubject(ctx, userID, subject)
	if err == nil && existing != nil {
		return &response.GenerateTaskResponse{
			TaskID:     existing.ID,
			Subject:    existing.Subject,
			TotalItems: existing.TotalItems,
			TaskMode:   existing.TaskMode,
			Status:     "already_exists",
		}, nil
	}

	// 获取用户该学科的知识树
	tree, err := s.treeRepo.GetTreeByUserAndSubject(ctx, userID, subject)
	if err != nil {
		return nil, fmt.Errorf("请先完成%s学科的初始化", subject)
	}

	// 五层出题算法选题
	selections, err := s.algorithm.SelectQuestions(ctx, userID, subject, tree.ID, 15)
	if err != nil {
		return nil, fmt.Errorf("出题算法执行失败: %w", err)
	}

	// 题型均衡
	selections = s.algorithm.EnsureQuestionTypeBalance(selections)

	// 根据选题结果构建题目
	var items []model.TaskItem
	now := time.Now()

	for i, sel := range selections {
		item := model.TaskItem{
			KnowledgeNodeID: &sel.KnowledgeNodeID,
			QuestionType:    sel.QuestionType,
			Difficulty:       sel.Difficulty,
			ItemMode:        sel.ItemMode,
			SortOrder:       i,
			Status:          1, // 待答
		}

		// 尝试从题库获取匹配题目
		if sel.QuestionID > 0 {
			// 错题重做
			item.QuestionID = &sel.QuestionID
			// 从错题表获取内容（简化处理，后续可补充）
			item.QuestionContent = fmt.Sprintf("错题#%d（复习）", sel.QuestionID)
		} else if sel.KnowledgeNodeID > 0 {
			// 从题库查找
			questions, qErr := s.taskRepo.GetQuestionsByKnowledgeNode(ctx, sel.KnowledgeNodeID, sel.QuestionType, 1)
			if qErr == nil && len(questions) > 0 {
				q := questions[0]
				item.QuestionID = &q.ID
				item.QuestionContent = q.Content
				item.Options = q.Options
				item.CorrectAnswer = q.Answer
				_ = s.taskRepo.IncrementQuestionUsage(ctx, q.ID)
			} else {
				// 题库不足，调用AI出题
				aiContent, aiErr := s.generateAIQuestion(ctx, sel, subject, tree)
				if aiErr != nil {
					zap.L().Warn("AI出题失败，使用占位题目",
						zap.Int64("node_id", sel.KnowledgeNodeID),
						zap.Error(aiErr),
					)
					item.QuestionContent = fmt.Sprintf("知识点#%d - %s题", sel.KnowledgeNodeID, sel.QuestionType)
				} else {
					item.QuestionContent = aiContent
					item.CorrectAnswer = "AI生成" // AI生成题目暂存占位答案
				}
			}
		}

		items = append(items, item)
	}

	// 创建每日任务记录
	task := &model.DailyTask{
		UserID:     userID,
		TaskDate:   now.Truncate(24 * time.Hour),
		Subject:    subject,
		TaskMode:   taskMode,
		Status:     1, // 待完成
		TotalItems: len(items),
	}
	if err := s.taskRepo.CreateTask(ctx, task); err != nil {
		return nil, fmt.Errorf("创建任务失败: %w", err)
	}

	// 回填TaskID并批量创建任务题目
	for i := range items {
		items[i].TaskID = task.ID
	}
	if err := s.taskRepo.CreateTaskItems(ctx, items); err != nil {
		return nil, fmt.Errorf("创建任务题目失败: %w", err)
	}

	zap.L().Info("每日任务生成完成",
		zap.Int64("user_id", userID),
		zap.Int64("task_id", task.ID),
		zap.String("subject", subject),
		zap.Int("item_count", len(items)),
	)

	return &response.GenerateTaskResponse{
		TaskID:     task.ID,
		Subject:    subject,
		TotalItems: len(items),
		TaskMode:   taskMode,
		Status:     "created",
	}, nil
}

// generateAIQuestion AI生成题目
func (s *TaskService) generateAIQuestion(ctx context.Context, sel QuestionSelection, subject string, tree *model.KnowledgeTree) (string, error) {
	// 获取知识点名称
	nodeName := "未知知识点"
	if sel.KnowledgeNodeID > 0 {
		node, err := s.treeRepo.GetNodeByID(ctx, sel.KnowledgeNodeID)
		if err == nil {
			nodeName = node.Name
		}
	}

	questionTypeMap := map[string]string{
		"choice": "选择题",
		"fill":   "填空题",
		"solve":  "解答题",
	}
	qtDesc := questionTypeMap[sel.QuestionType]
	if qtDesc == "" {
		qtDesc = "选择题"
	}

	difficulty := int(sel.Difficulty)
	if difficulty <= 0 {
		difficulty = 2
	}

	content, err := s.llmClient.GenerateQuestions(ctx, nodeName, qtDesc, difficulty, subject, int(tree.Grade), 1)
	if err != nil {
		return "", fmt.Errorf("AI出题失败: %w", err)
	}

	return content, nil
}

// GetTodayTask 获取今日任务
func (s *TaskService) GetTodayTask(ctx context.Context, userID int64, subject string) (*response.TaskDetailResponse, error) {
	task, err := s.taskRepo.GetTodayTaskByUserAndSubject(ctx, userID, subject)
	if err != nil {
		return nil, fmt.Errorf("今日暂无%s任务", subject)
	}

	return s.buildTaskDetail(ctx, task)
}

// GetTaskHistory 获取任务历史
func (s *TaskService) GetTaskHistory(ctx context.Context, userID int64, subject string, status *int16, page, pageSize int) ([]response.TaskListResponse, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	tasks, total, err := s.taskRepo.ListTasksByUser(ctx, userID, subject, status, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询任务历史失败: %w", err)
	}

	result := make([]response.TaskListResponse, 0, len(tasks))
	for _, t := range tasks {
		result = append(result, response.TaskListResponse{
			ID:             t.ID,
			Subject:        t.Subject,
			TaskDate:       t.TaskDate.Format("2006-01-02"),
			TaskMode:       t.TaskMode,
			Status:         t.Status,
			TotalItems:     t.TotalItems,
			CompletedItems: t.CompletedItems,
			CorrectItems:   t.CorrectItems,
			TimeLimitMin:   t.TimeLimitMin,
			CreatedAt:      t.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return result, total, nil
}

// GetTaskDetail 获取任务详情
func (s *TaskService) GetTaskDetail(ctx context.Context, taskID int64) (*response.TaskDetailResponse, error) {
	task, err := s.taskRepo.GetTaskByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("任务不存在")
	}
	return s.buildTaskDetail(ctx, task)
}

// StartTask 开始任务
func (s *TaskService) StartTask(ctx context.Context, taskID int64) error {
	task, err := s.taskRepo.GetTaskByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("任务不存在")
	}

	if task.Status != 1 {
		return fmt.Errorf("任务状态不允许开始")
	}

	now := time.Now()
	task.Status = 2 // 进行中
	task.StartAt = &now
	return s.taskRepo.UpdateTask(ctx, task)
}

// SubmitAnswer 提交答案
func (s *TaskService) SubmitAnswer(ctx context.Context, taskItemID int64, answer string, answerDuration *int) (*response.SubmitAnswerResponse, error) {
	item, err := s.taskRepo.GetTaskItemByID(ctx, taskItemID)
	if err != nil {
		return nil, fmt.Errorf("题目不存在")
	}

	if item.Status != 1 {
		return nil, fmt.Errorf("该题目已作答")
	}

	// 更新答案
	item.StudentAnswer = answer
	item.AnswerDuration = answerDuration
	item.Status = 2 // 已答

	// 即时判题（选择题和填空题可以即时判对错）
	var isCorrect *bool
	var score *float64

	if item.QuestionType == "choice" || item.QuestionType == "fill" {
		correct := answer == item.CorrectAnswer
		isCorrect = &correct
		item.IsCorrect = isCorrect

		s := 0.0
		if correct {
			s = 10.0 // 每题10分
		}
		score = &s
		item.Score = score
		item.Status = 3 // 已批改
	}

	if err := s.taskRepo.UpdateTaskItem(ctx, item); err != nil {
		return nil, fmt.Errorf("更新答题结果失败: %w", err)
	}

	// 记录答题历史
	if item.QuestionID != nil && *item.QuestionID > 0 {
		history := &model.UserQuestionHistory{
			UserID:     0, // TODO: 从context获取
			QuestionID: *item.QuestionID,
			TaskItemID: &item.ID,
			IsCorrect:  isCorrect,
			AnsweredAt: time.Now(),
		}
		_ = s.taskRepo.RecordUserHistory(ctx, history)
	}

	// 更新任务进度
	s.updateTaskProgress(ctx, item.TaskID)

	return &response.SubmitAnswerResponse{
		TaskItemID: item.ID,
		IsCorrect:  isCorrect,
		Score:      score,
	}, nil
}

// FinishTask 完成任务
func (s *TaskService) FinishTask(ctx context.Context, taskID int64) error {
	task, err := s.taskRepo.GetTaskByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("任务不存在")
	}

	if task.Status != 2 {
		return fmt.Errorf("任务状态不允许完成")
	}

	now := time.Now()
	task.Status = 3 // 已完成
	task.FinishAt = &now

	// 计算实际用时（分钟）
	if task.StartAt != nil {
		duration := int(now.Sub(*task.StartAt).Minutes())
		task.ActualTimeMin = &duration
	}

	if err := s.taskRepo.UpdateTask(ctx, task); err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	// 批改未批改的题目（解答题用AI批改）
	items, err := s.taskRepo.GetTaskItemsByTaskID(ctx, taskID)
	if err != nil {
		return nil // 不阻断完成流程
	}

	for _, item := range items {
		if item.Status == 2 && item.QuestionType == "solve" {
			// AI批改解答题
			s.aiGradeItem(ctx, &item)
			_ = s.taskRepo.UpdateTaskItem(ctx, &item)
		}
	}

	// 更新知识节点掌握率
	s.updateMasteryRates(ctx, task)

	// 更新错题库
	s.updateErrorQuestions(ctx, task, items)

	zap.L().Info("任务完成",
		zap.Int64("task_id", taskID),
		zap.Int("total", task.TotalItems),
		zap.Int("correct", task.CorrectItems),
	)

	return nil
}

// updateTaskProgress 更新任务进度
func (s *TaskService) updateTaskProgress(ctx context.Context, taskID int64) {
	items, err := s.taskRepo.GetTaskItemsByTaskID(ctx, taskID)
	if err != nil {
		return
	}

	completed := 0
	correct := 0
	for _, item := range items {
		if item.Status >= 2 {
			completed++
		}
		if item.IsCorrect != nil && *item.IsCorrect {
			correct++
		}
	}

	task, err := s.taskRepo.GetTaskByID(ctx, taskID)
	if err != nil {
		return
	}
	task.CompletedItems = completed
	task.CorrectItems = correct
	_ = s.taskRepo.UpdateTask(ctx, task)
}

// aiGradeItem AI批改解答题
func (s *TaskService) aiGradeItem(ctx context.Context, item *model.TaskItem) {
	if s.llmClient == nil {
		return
	}

	result, err := s.llmClient.GradeAnswer(ctx, item.QuestionContent, item.CorrectAnswer, item.StudentAnswer, "")
	if err != nil {
		zap.L().Warn("AI批改失败", zap.Int64("item_id", item.ID), zap.Error(err))
		return
	}

	item.IsCorrect = &result.IsCorrect
	item.Score = &result.TotalScore
	item.Status = 3 // 已批改

	// 存储批改结果
	gradingJSON, _ := json.Marshal(result)
	item.GradingResult = gradingJSON
}

// updateMasteryRates 更新知识节点掌握率
func (s *TaskService) updateMasteryRates(ctx context.Context, task *model.DailyTask) {
	items, err := s.taskRepo.GetTaskItemsByTaskID(ctx, task.ID)
	if err != nil {
		return
	}

	// 按知识点节点分组统计
	nodeStats := make(map[int64]struct{ correct, total int })
	for _, item := range items {
		if item.KnowledgeNodeID == nil {
			continue
		}
		stats := nodeStats[*item.KnowledgeNodeID]
		stats.total++
		if item.IsCorrect != nil && *item.IsCorrect {
			stats.correct++
		}
		nodeStats[*item.KnowledgeNodeID] = stats
	}

	// 更新Level 5节点的掌握率
	for nodeID, stats := range nodeStats {
		if stats.total == 0 {
			continue
		}
		node, err := s.treeRepo.GetNodeByID(ctx, nodeID)
		if err != nil {
			continue
		}

		// 更新统计
		newCorrect := node.CorrectCount + stats.correct
		newTotal := node.QuestionCount + stats.total
		newRate := float64(newCorrect) / float64(newTotal) * 100

		_ = s.treeRepo.UpdateNodeMastery(ctx, nodeID, newRate)

		// 更新计数和最后练习时间
		now := time.Now()
		s.updateNodeCounts(ctx, nodeID, newCorrect, newTotal, &now)
	}

	// 级联更新父节点掌握率
	s.cascadeMasteryUpdate(ctx, task.Subject, task.UserID)
}

// updateNodeCounts 更新节点答题计数
func (s *TaskService) updateNodeCounts(ctx context.Context, nodeID int64, correctCount, questionCount int, lastPracticed *time.Time) {
	// 通过直接SQL更新（避免GORM零值问题）
	node, err := s.treeRepo.GetNodeByID(ctx, nodeID)
	if err != nil {
		return
	}
	node.CorrectCount = correctCount
	node.QuestionCount = questionCount
	node.LastPracticed = lastPracticed
	_ = s.treeRepo.DB().WithContext(ctx).Save(node).Error
}

// cascadeMasteryUpdate 级联更新掌握率（Level 5 → Level 4 → Level 3 → Level 2）
func (s *TaskService) cascadeMasteryUpdate(ctx context.Context, subject string, userID int64) {
	tree, err := s.treeRepo.GetTreeByUserAndSubject(ctx, userID, subject)
	if err != nil {
		return
	}

	// 从Level 5向上级联
	for level := int16(4); level >= 2; level-- {
		parentNodes, err := s.treeRepo.GetNodesByTreeIDAndLevel(ctx, tree.ID, level)
		if err != nil {
			continue
		}

		for _, parent := range parentNodes {
			// 获取子节点
			var children []model.KnowledgeNode
			s.treeRepo.DB().WithContext(ctx).
				Where("tree_id = ? AND parent_id = ?", tree.ID, parent.ID).
				Find(&children)

			if len(children) == 0 {
				continue
			}

			var totalMastery float64
			for _, child := range children {
				totalMastery += child.MasteryRate
			}
			avgMastery := totalMastery / float64(len(children))

			_ = s.treeRepo.UpdateNodeMastery(ctx, parent.ID, avgMastery)
		}
	}
}

// updateErrorQuestions 更新错题库
func (s *TaskService) updateErrorQuestions(ctx context.Context, task *model.DailyTask, items []model.TaskItem) {
	for _, item := range items {
		if item.IsCorrect == nil || *item.IsCorrect {
			continue // 只记录错题
		}

		errorType := "wrong"
		if item.StudentAnswer == "" {
			errorType = "blank"
		}

		errorQ := &model.ErrorQuestion{
			UserID:          task.UserID,
			Subject:         task.Subject,
			KnowledgeNodeID: item.KnowledgeNodeID,
			QuestionType:    item.QuestionType,
			QuestionContent: item.QuestionContent,
			CorrectAnswer:   item.CorrectAnswer,
			StudentAnswer:   item.StudentAnswer,
			ErrorType:       errorType,
			SourceType:      "daily_task",
			SourceID:        &task.ID,
			Difficulty:      item.Difficulty,
		}

		// 检查是否已存在相同错题
		// 简化处理：直接创建
		s.taskRepo.DB().WithContext(ctx).Create(errorQ)
	}
}

// buildTaskDetail 构建任务详情响应
func (s *TaskService) buildTaskDetail(ctx context.Context, task *model.DailyTask) (*response.TaskDetailResponse, error) {
	items, err := s.taskRepo.GetTaskItemsByTaskID(ctx, task.ID)
	if err != nil {
		return nil, fmt.Errorf("查询任务题目失败: %w", err)
	}

	itemResponses := make([]response.TaskItemResponse, 0, len(items))
	for _, item := range items {
		itemResponses = append(itemResponses, response.TaskItemResponse{
			ID:              item.ID,
			QuestionID:      item.QuestionID,
			KnowledgeNodeID: item.KnowledgeNodeID,
			QuestionType:    item.QuestionType,
			QuestionContent: item.QuestionContent,
			Options:         item.Options,
			Difficulty:      item.Difficulty,
			ItemMode:        item.ItemMode,
			SortOrder:       item.SortOrder,
			Status:          item.Status,
			StudentAnswer:   item.StudentAnswer,
			IsCorrect:       item.IsCorrect,
			Score:           item.Score,
		})
	}

	return &response.TaskDetailResponse{
		ID:             task.ID,
		Subject:        task.Subject,
		TaskDate:       task.TaskDate.Format("2006-01-02"),
		TaskMode:       task.TaskMode,
		Status:         task.Status,
		TotalItems:     task.TotalItems,
		CompletedItems: task.CompletedItems,
		CorrectItems:   task.CorrectItems,
		TimeLimitMin:   task.TimeLimitMin,
		ActualTimeMin:  task.ActualTimeMin,
		PDFUrl:         task.PDFUrl,
		Items:          itemResponses,
		CreatedAt:      task.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

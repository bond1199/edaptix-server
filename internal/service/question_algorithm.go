package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/edaptix/server/internal/repository"
	"go.uber.org/zap"
)

// AbilityLevel 能力层级
type AbilityLevel string

const (
	AbilityBasic    AbilityLevel = "basic"    // 基础层：掌握率 < 0.4
	AbilityImproving AbilityLevel = "improving" // 提升层：掌握率 0.4-0.7
	AbilityAdvanced AbilityLevel = "advanced" // 冲刺层：掌握率 > 0.7
)

// QuestionSelection 出题算法选中的题目
type QuestionSelection struct {
	QuestionID       int64
	KnowledgeNodeID  int64
	QuestionType     string
	Difficulty       int16
	ItemMode         string // remedial(补弱) / advanced(拔高)
	Reason           string // 选中原因
}

// QuestionAlgorithm 五层出题算法
type QuestionAlgorithm struct {
	taskRepo *repository.TaskRepo
	treeRepo *repository.KnowledgeTreeRepo
}

// NewQuestionAlgorithm 创建出题算法
func NewQuestionAlgorithm(taskRepo *repository.TaskRepo, treeRepo *repository.KnowledgeTreeRepo) *QuestionAlgorithm {
	return &QuestionAlgorithm{
		taskRepo: taskRepo,
		treeRepo: treeRepo,
	}
}

// CalculateAbilityLevel 计算学生能力层级
func (a *QuestionAlgorithm) CalculateAbilityLevel(ctx context.Context, treeID int64) (AbilityLevel, float64, error) {
	nodes, err := a.treeRepo.GetNodesByTreeIDAndLevel(ctx, treeID, 5)
	if err != nil {
		return AbilityBasic, 0, fmt.Errorf("查询知识点节点失败: %w", err)
	}

	if len(nodes) == 0 {
		return AbilityBasic, 0, nil
	}

	var totalMastery float64
	for _, node := range nodes {
		totalMastery += node.MasteryRate
	}
	avgMastery := totalMastery / float64(len(nodes))

	switch {
	case avgMastery < 0.4:
		return AbilityBasic, avgMastery, nil
	case avgMastery <= 0.7:
		return AbilityImproving, avgMastery, nil
	default:
		return AbilityAdvanced, avgMastery, nil
	}
}

// GetRemedialAdvancedRatio 根据能力层级获取补弱/拔高比例
func (a *QuestionAlgorithm) GetRemedialAdvancedRatio(level AbilityLevel) (remedialRatio, advancedRatio float64) {
	switch level {
	case AbilityBasic:
		return 0.8, 0.2
	case AbilityImproving:
		return 0.6, 0.4
	case AbilityAdvanced:
		return 0.4, 0.6
	default:
		return 0.6, 0.4
	}
}

// SelectQuestions 五层出题算法主入口
// totalQuestions: 需要出多少道题
// subject: 学科
// treeID: 知识树ID
func (a *QuestionAlgorithm) SelectQuestions(ctx context.Context, userID int64, subject string, treeID int64, totalQuestions int) ([]QuestionSelection, error) {
	if totalQuestions <= 0 {
		totalQuestions = 15 // 默认15题
	}

	// 计算能力层级
	level, avgMastery, err := a.CalculateAbilityLevel(ctx, treeID)
	if err != nil {
		return nil, err
	}

	zap.L().Info("出题算法-能力分层",
		zap.Int64("user_id", userID),
		zap.String("subject", subject),
		zap.String("level", string(level)),
		zap.Float64("avg_mastery", avgMastery),
	)

	remedialRatio, _ := a.GetRemedialAdvancedRatio(level)
	remedialCount := int(math.Round(float64(totalQuestions) * remedialRatio))
	advancedCount := totalQuestions - remedialCount

	// Layer 1: 错题匹配 - 获取用户未解决的错题
	errorSelections := a.selectFromErrors(ctx, userID, subject, remedialCount)

	// Layer 2: 薄弱权重 - 优先出薄弱知识点题目
	weakSelections := a.selectFromWeakNodes(ctx, treeID, remedialCount-len(errorSelections))

	// Layer 3: 进度锁定 - 已掌握节点降低权重（通过薄弱权重层自然过滤）
	// 已在 selectFromWeakNodes 中通过 mastery_rate < 0.8 条件过滤

	// Layer 4: 遗忘曲线 - 优先出较久未练习的知识点
	reviewSelections := a.selectFromForgettingCurve(ctx, treeID)

	// Layer 5: 题型均衡 - 补足剩余题目
	balancedSelections := a.selectForBalance(ctx, treeID, totalQuestions, subject)

	// 合并所有选择（去重）
	merged := a.mergeSelections(errorSelections, weakSelections, reviewSelections, balancedSelections, totalQuestions, remedialCount, advancedCount)

	zap.L().Info("出题算法-选题完成",
		zap.Int64("user_id", userID),
		zap.String("subject", subject),
		zap.Int("total_selected", len(merged)),
		zap.Int("error_count", len(errorSelections)),
		zap.Int("weak_count", len(weakSelections)),
	)

	return merged, nil
}

// Layer 1: 错题匹配
func (a *QuestionAlgorithm) selectFromErrors(ctx context.Context, userID int64, subject string, maxCount int) []QuestionSelection {
	if maxCount <= 0 {
		return nil
	}

	errorIDs, err := a.taskRepo.GetErrorQuestionIDsByUser(ctx, userID, subject)
	if err != nil {
		zap.L().Warn("出题算法-获取错题失败", zap.Error(err))
		return nil
	}

	var selections []QuestionSelection
	for i, id := range errorIDs {
		if i >= maxCount {
			break
		}
		selections = append(selections, QuestionSelection{
			QuestionID:  id,
			ItemMode:    "remedial",
			Reason:      "错题复习",
			Difficulty:  2,
			QuestionType: "choice", // 默认选择题
		})
	}
	return selections
}

// Layer 2: 薄弱权重
func (a *QuestionAlgorithm) selectFromWeakNodes(ctx context.Context, treeID int64, maxCount int) []QuestionSelection {
	if maxCount <= 0 {
		return nil
	}

	// 获取掌握率 < 0.6 的薄弱知识点
	weakNodes, err := a.taskRepo.GetWeakNodes(ctx, treeID, 0.6)
	if err != nil {
		zap.L().Warn("出题算法-获取薄弱节点失败", zap.Error(err))
		return nil
	}

	var selections []QuestionSelection
	for i, node := range weakNodes {
		if i >= maxCount {
			break
		}
		selections = append(selections, QuestionSelection{
			KnowledgeNodeID: node.ID,
			ItemMode:        "remedial",
			Reason:          fmt.Sprintf("薄弱知识点(掌握率%.0f%%)", node.MasteryRate),
			Difficulty:       2,
		})
	}
	return selections
}

// Layer 4: 遗忘曲线
func (a *QuestionAlgorithm) selectFromForgettingCurve(ctx context.Context, treeID int64) []QuestionSelection {
	// 3天前练习过的知识点需要复习
	reviewBefore := time.Now().AddDate(0, 0, -3)
	nodes, err := a.taskRepo.GetNodesForReview(ctx, treeID, reviewBefore)
	if err != nil {
		zap.L().Warn("出题算法-获取复习节点失败", zap.Error(err))
		return nil
	}

	var selections []QuestionSelection
	for _, node := range nodes {
		selections = append(selections, QuestionSelection{
			KnowledgeNodeID: node.ID,
			ItemMode:        "remedial",
			Reason:          "遗忘曲线复习",
			Difficulty:       2,
		})
	}
	return selections
}

// Layer 5: 题型均衡
func (a *QuestionAlgorithm) selectForBalance(ctx context.Context, treeID int64, totalQuestions int, subject string) []QuestionSelection {
	// 获取未练习的节点
	unpracticed, err := a.taskRepo.GetUnpracticedNodes(ctx, treeID)
	if err != nil {
		zap.L().Warn("出题算法-获取未练习节点失败", zap.Error(err))
		return nil
	}

	var selections []QuestionSelection
	for i, node := range unpracticed {
		if i >= totalQuestions {
			break
		}
		selections = append(selections, QuestionSelection{
			KnowledgeNodeID: node.ID,
			ItemMode:        "advanced",
			Reason:          "新知识点拓展",
			Difficulty:       3,
		})
	}
	return selections
}

// mergeSelections 合并选题并去重
func (a *QuestionAlgorithm) mergeSelections(errorSels, weakSels, reviewSels, balanceSels []QuestionSelection, totalQuestions, remedialCount, advancedCount int) []QuestionSelection {
	seen := make(map[int64]bool) // 用于去重（按KnowledgeNodeID和QuestionID）
	var result []QuestionSelection

	// 优先级：错题 > 薄弱 > 遗忘 > 均衡
	addSelection := func(sel QuestionSelection) {
		key := sel.QuestionID
		if key == 0 {
			key = -sel.KnowledgeNodeID // 负数区分知识点ID
		}
		if seen[key] {
			return
		}
		seen[key] = true
		result = append(result, sel)
	}

	// 先添加补弱题目
	for _, sel := range errorSels {
		if len(result) >= remedialCount {
			break
		}
		addSelection(sel)
	}
	for _, sel := range weakSels {
		if len(result) >= remedialCount {
			break
		}
		addSelection(sel)
	}
	for _, sel := range reviewSels {
		if len(result) >= remedialCount {
			break
		}
		addSelection(sel)
	}

	// 再添加拔高题目
	for _, sel := range balanceSels {
		if len(result) >= totalQuestions {
			break
		}
		addSelection(sel)
	}

	// 截断到总数
	if len(result) > totalQuestions {
		result = result[:totalQuestions]
	}

	return result
}

// EnsureQuestionTypeBalance 确保题型均衡（4:3:3 → 选择:填空:解答）
// 修改selections中的QuestionType字段
func (a *QuestionAlgorithm) EnsureQuestionTypeBalance(selections []QuestionSelection) []QuestionSelection {
	n := len(selections)
	if n == 0 {
		return selections
	}

	choiceCount := int(math.Round(float64(n) * 0.4))
	fillCount := int(math.Round(float64(n) * 0.3))
	solveCount := n - choiceCount - fillCount

	questionTypes := []string{"choice", "fill", "solve"}
	counts := []int{choiceCount, fillCount, solveCount}

	idx := 0
	for i, qt := range questionTypes {
		for j := 0; j < counts[i] && idx < n; j++ {
			selections[idx].QuestionType = qt
			idx++
		}
	}

	return selections
}

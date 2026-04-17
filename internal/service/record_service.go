package service

import (
	"context"
	"errors"
	"strings"

	"Project_sekai_search/internal/dto"
	"Project_sekai_search/internal/model"
	"Project_sekai_search/internal/pkg/errcode"
	"Project_sekai_search/internal/repository"

	"gorm.io/gorm"
)

type RecordService struct {
	recordRepo repository.RecordRepository
}

func NewRecordService(recordRepo repository.RecordRepository) *RecordService {
	return &RecordService{recordRepo: recordRepo}
}

func (s *RecordService) UploadRecord(ctx context.Context, userID uint, req dto.UploadRecordRequest) (*model.Record, bool, float64, error) {
	difficulty, err := s.recordRepo.GetMusicDifficultyByID(ctx, req.MusicDifficultyID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, 0, errcode.ErrBadRequest
		}
		return nil, false, 0, errcode.ErrInternal
	}

	achievement, err := s.recordRepo.GetMusicAchievementByID(ctx, req.MusicAchievementID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, 0, errcode.ErrBadRequest
		}
		return nil, false, 0, errcode.ErrInternal
	}
	diffType := strings.ToLower(strings.TrimSpace(difficulty.MusicDifficulty))
	achType := strings.ToLower(strings.TrimSpace(achievement.MusicDifficultyType))
	if achType != diffType {
		return nil, false, 0, errcode.ErrBadRequest
	}

	scoreValue := calculateRecordScore(float64(difficulty.PlayLevel), achievement.Status)
	record, created, avgB30, err := s.recordRepo.UpsertUserRecordAndRefreshB30(ctx, userID, difficulty, req.MusicAchievementID, scoreValue)
	if err != nil {
		return nil, false, 0, errcode.ErrInternal
	}

	return record, created, avgB30, nil
}

func (s *RecordService) GetBest30(ctx context.Context, userID uint, calcMode string) ([]dto.Best30Item, float64, error) {
	items, avgB30, err := s.recordRepo.GetBest30ByUserID(ctx, userID, normalizeCalcMode(calcMode))
	if err != nil {
		return nil, 0, errcode.ErrInternal
	}

	return items, avgB30, nil
}

func (s *RecordService) GetB30Trend(ctx context.Context, userID uint, calcMode string) ([]dto.B30TrendPoint, error) {
	items, err := s.recordRepo.GetB30TrendByUserID(ctx, userID, normalizeCalcMode(calcMode))
	if err != nil {
		return nil, errcode.ErrInternal
	}
	return items, nil
}

func (s *RecordService) GetStatistics(ctx context.Context, userID uint, difficulty string, mode string, minLevel uint, maxLevel uint) (*dto.RecordStatisticsResponse, error) {
	diff := normalizeStatisticsDifficulty(difficulty)
	m := normalizeStatisticsMode(mode)
	minLevel, maxLevel = normalizeStatisticsLevelRange(minLevel, maxLevel)

	resp := &dto.RecordStatisticsResponse{
		Difficulty: diff,
		Mode:       m,
		MinLevel:   minLevel,
		MaxLevel:   maxLevel,
		Buckets:    make([]dto.RecordStatisticsBucket, 0),
	}

	if m == "by_global_level" {
		resp.Difficulty = "all"
		items, err := s.recordRepo.ListGlobalStatisticsByLevelRange(ctx, userID, minLevel, maxLevel)
		if err != nil {
			return nil, errcode.ErrInternal
		}
		byLevel := make(map[uint]dto.RecordStatisticsBucket, len(items))
		for _, it := range items {
			byLevel[it.PlayLevel] = it
		}
		resp.Buckets = make([]dto.RecordStatisticsBucket, 0, int(maxLevel-minLevel+1))
		for lv := minLevel; lv <= maxLevel; lv++ {
			if it, ok := byLevel[lv]; ok {
				resp.Buckets = append(resp.Buckets, it)
				continue
			}
			resp.Buckets = append(resp.Buckets, dto.RecordStatisticsBucket{
				PlayLevel: lv,
			})
		}
		for _, it := range resp.Buckets {
			resp.TotalCharts += it.TotalCharts
		}
		return resp, nil
	}

	if m == "by_level" {
		items, err := s.recordRepo.ListDifficultyStatisticsByLevel(ctx, userID, diff)
		if err != nil {
			return nil, errcode.ErrInternal
		}
		resp.Buckets = items
		for _, it := range items {
			resp.TotalCharts += it.TotalCharts
		}
		return resp, nil
	}

	overview, err := s.recordRepo.GetDifficultyStatisticsOverview(ctx, userID, diff)
	if err != nil {
		return nil, errcode.ErrInternal
	}
	if overview != nil {
		resp.Buckets = append(resp.Buckets, *overview)
		resp.TotalCharts = overview.TotalCharts
	}
	return resp, nil
}

func (s *RecordService) GetUserRecordStatuses(ctx context.Context, userID uint) ([]dto.UserRecordStatusItem, error) {
	items, err := s.recordRepo.ListUserRecordStatuses(ctx, userID)
	if err != nil {
		return nil, errcode.ErrInternal
	}
	return items, nil
}

func (s *RecordService) GetAchievementMap(ctx context.Context) (map[string]map[string]uint, error) {
	items, err := s.recordRepo.ListMusicAchievements(ctx)
	if err != nil {
		return nil, errcode.ErrInternal
	}

	result := make(map[string]map[string]uint)
	for _, it := range items {
		diffType := strings.ToLower(strings.TrimSpace(it.MusicDifficultyType))
		if diffType == "" {
			continue
		}
		if _, ok := result[diffType]; !ok {
			result[diffType] = map[string]uint{
				"clear":       0,
				"full_combo":  0,
				"all_perfect": 0,
			}
		}
		status := strings.ToLower(strings.TrimSpace(it.Status))
		switch {
		case strings.Contains(status, "all_perfect") || strings.Contains(status, "all perfect") || status == "ap":
			if result[diffType]["all_perfect"] == 0 {
				result[diffType]["all_perfect"] = it.ID
			}
		case strings.Contains(status, "full_combo") || strings.Contains(status, "full combo") || status == "fc":
			if result[diffType]["full_combo"] == 0 {
				result[diffType]["full_combo"] = it.ID
			}
		case strings.Contains(status, "clear"):
			if result[diffType]["clear"] == 0 {
				result[diffType]["clear"] = it.ID
			}
		}
	}

	return result, nil
}

func (s *RecordService) DeleteRecord(ctx context.Context, userID uint, req dto.DeleteRecordRequest) (bool, float64, error) {
	_, err := s.recordRepo.GetMusicDifficultyByID(ctx, req.MusicDifficultyID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, 0, errcode.ErrBadRequest
		}
		return false, 0, errcode.ErrInternal
	}

	deleted, avgB30, err := s.recordRepo.DeleteUserRecordByDifficultyAndRefreshB30(ctx, userID, req.MusicDifficultyID)
	if err != nil {
		return false, 0, errcode.ErrInternal
	}
	return deleted, avgB30, nil
}

func calculateRecordScore(level float64, achievementStatus string) float64 {
	status := strings.ToLower(strings.TrimSpace(achievementStatus))

	// AP: same as level
	if strings.Contains(status, "all_perfect") || strings.Contains(status, "all perfect") || strings.Contains(status, "ap") {
		return level
	}

	// FC: level - 1.5, but level>=33 is level - 1
	if strings.Contains(status, "full_combo") || strings.Contains(status, "full combo") || strings.Contains(status, "fc") {
		if level >= 33 {
			return level - 1.0
		}
		return level - 1.5
	}

	// Fallback for non-FC/AP statuses
	return level - 5.0
}

func normalizeCalcMode(raw string) string {
	if strings.EqualFold(strings.TrimSpace(raw), "const") {
		return "const"
	}
	return "official"
}

func normalizeStatisticsDifficulty(raw string) string {
	diff := strings.ToLower(strings.TrimSpace(raw))
	switch diff {
	case "easy", "normal", "hard", "expert", "master", "append":
		return diff
	case "matser":
		return "master"
	default:
		return "master"
	}
}

func normalizeStatisticsMode(raw string) string {
	mode := strings.ToLower(strings.TrimSpace(raw))
	switch mode {
	case "by_level", "difficulty", "detail":
		return "by_level"
	case "by_global_level", "global_level", "all_level", "level_range":
		return "by_global_level"
	default:
		return "by_difficulty"
	}
}

func normalizeStatisticsLevelRange(minLevel uint, maxLevel uint) (uint, uint) {
	if minLevel == 0 {
		minLevel = 1
	}
	if maxLevel == 0 {
		maxLevel = 40
	}
	if minLevel > 40 {
		minLevel = 40
	}
	if maxLevel > 40 {
		maxLevel = 40
	}
	if minLevel > maxLevel {
		minLevel, maxLevel = maxLevel, minLevel
	}
	return minLevel, maxLevel
}

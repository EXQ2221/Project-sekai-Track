package service

import (
	"Project_sekai_search/internal/dto"
	"Project_sekai_search/internal/model"
	"Project_sekai_search/internal/pkg/errcode"
	"Project_sekai_search/internal/repository"
	"context"
	"errors"
	"math"
	"math/rand"
	"strings"

	"gorm.io/gorm"
)

const (
	allPerfectMode = 0
	fullComboMode  = 1
	calcModeConst  = "const"
	calcModeOff    = "official"
)

type RandomService struct {
	userRepo   repository.UserRepository
	musicRepo  repository.MusicRepository
	recordRepo repository.RecordRepository
}

func NewRandomService(
	userRepo repository.UserRepository,
	musicRepo repository.MusicRepository,
	recordRepo repository.RecordRepository,
) *RandomService {
	return &RandomService{
		userRepo:   userRepo,
		musicRepo:  musicRepo,
		recordRepo: recordRepo,
	}
}

func (s *RandomService) RandomMusicRecommendation(ctx context.Context, userID uint, calcMode string) (*dto.RandomResponse, error) {
	var user model.User

	var err error
	calcMode, err = normalizeRandomCalcMode(calcMode)
	if err != nil {
		return nil, err
	}

	err = s.userRepo.FindUserByID(ctx, userID, &user)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrBadRequest
		}
		return nil, errcode.ErrInternal
	}

	b30Avg := user.B30Avg
	if calcMode == calcModeConst {
		b30Avg = user.B30AvgConst
	}
	actualMode := calcMode

	userAchievementRanks, err := s.loadUserAchievementRanks(ctx, userID)
	if err != nil {
		return nil, errcode.ErrInternal
	}

	mode := randomPickMode()
	targetRank := targetAchievementRank(mode)
	target := pickTargetLevelOrConst(b30Avg, actualMode, mode)

	diffs, err := s.pickCandidateDifficulties(ctx, actualMode, target, userAchievementRanks, targetRank)
	if err != nil {
		return nil, errcode.ErrInternal
	}
	if len(diffs) == 0 && calcMode == calcModeConst {
		// const candidates are sparse; degrade to official mode when needed.
		actualMode = calcModeOff
		target = pickTargetLevelOrConst(user.B30Avg, actualMode, mode)
		diffs, err = s.pickCandidateDifficulties(ctx, actualMode, target, userAchievementRanks, targetRank)
		if err != nil {
			return nil, errcode.ErrInternal
		}
	}
	if len(diffs) == 0 {
		return nil, errcode.ErrNotFound
	}

	targetDiff := diffs[rand.Intn(len(diffs))]
	music, err := s.musicRepo.GetMusicByID(ctx, targetDiff.MusicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrNotFound
		}
		return nil, errcode.ErrInternal
	}

	resp := &dto.RandomResponse{
		SongID:            music.ID,
		AssetBundleName:   music.AssetBundleName,
		Title:             music.Title,
		MusicDifficultyID: targetDiff.ID,
		MusicDifficulty:   strings.ToLower(strings.TrimSpace(targetDiff.MusicDifficulty)),
		PlayLevel:         targetDiff.PlayLevel,
		ConstValue:        targetDiff.Const,
		CalcMode:          actualMode,
		Type:              recommendationTypeText(mode),
		UserAchievement:   "not_played",
	}

	record, err := s.recordRepo.FindRecordByDiffID(ctx, userID, targetDiff.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, nil
		}
		return nil, errcode.ErrInternal
	}

	resp.UserAchievementID = record.MusicAchievementID
	resp.UserScoreValue = record.ScoreValue
	achievement, err := s.recordRepo.GetMusicAchievementByID(ctx, record.MusicAchievementID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, nil
		}
		return nil, errcode.ErrInternal
	}
	resp.UserAchievement = normalizeAchievementText(achievement.Status)
	return resp, nil
}

func randomPickMode() int {
	if rand.Intn(2) == 0 {
		return allPerfectMode
	}
	return fullComboMode
}

func randFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func randInt(n int) int {
	if n < 0 {
		n = 0
	}
	a := n + 1
	b := n + 2

	if rand.Intn(2) == 0 {
		return a
	}
	return b
}

func normalizeRandomCalcMode(raw string) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(raw))
	switch mode {
	case calcModeConst:
		return calcModeConst, nil
	case calcModeOff:
		return calcModeOff, nil
	default:
		return "", errcode.ErrBadRequest
	}
}

func pickTargetLevelOrConst(b30Avg float64, calcMode string, targetType int) float64 {
	if calcMode == calcModeOff {
		b30Avg = math.Round(b30Avg)
	}

	base := b30Avg
	if targetType == fullComboMode {
		if calcMode == calcModeOff {
			base = b30Avg + 1.0
		} else {
			base = b30Avg + 1.5
		}
	}

	if calcMode == calcModeOff {
		return float64(randInt(int(base)))
	}
	return roundToOneDecimal(randFloat(base-1.5, base+1.5))
}

func (s *RandomService) pickCandidateDifficulties(
	ctx context.Context,
	calcMode string,
	target float64,
	userAchievementRanks map[uint]int,
	targetRank int,
) ([]model.MusicDifficulty, error) {
	levelOffsets := []int{0, 1, -1, 2, -2}
	if calcMode == calcModeConst {
		constOffsets := make([]float64, 0, 31)
		constOffsets = append(constOffsets, 0)
		for i := 1; i <= 15; i++ {
			offset := float64(i) * 0.1
			constOffsets = append(constOffsets, offset, -offset)
		}
		seenConst := make(map[float64]struct{}, len(constOffsets))
		for _, offset := range constOffsets {
			candidate := roundToOneDecimal(target + offset)
			if candidate <= 0 {
				continue
			}
			if _, ok := seenConst[candidate]; ok {
				continue
			}
			seenConst[candidate] = struct{}{}
			diffs, err := s.musicRepo.FindDifficultiesByConst(ctx, candidate)
			if err != nil {
				return nil, err
			}
			eligible := filterDifficultiesByTargetRank(diffs, userAchievementRanks, targetRank)
			if len(eligible) > 0 {
				return eligible, nil
			}
		}
	}

	baseLevel := int(math.Round(target))
	if baseLevel < 1 {
		baseLevel = 1
	}
	seenLevel := make(map[uint]struct{}, len(levelOffsets))
	for _, offset := range levelOffsets {
		level := baseLevel + offset
		if level < 1 {
			continue
		}
		lv := uint(level)
		if _, ok := seenLevel[lv]; ok {
			continue
		}
		seenLevel[lv] = struct{}{}
		diffs, err := s.musicRepo.FindDifficultiesByLevel(ctx, lv)
		if err != nil {
			return nil, err
		}
		eligible := filterDifficultiesByTargetRank(diffs, userAchievementRanks, targetRank)
		if len(eligible) > 0 {
			return eligible, nil
		}
	}

	return nil, nil
}

func roundToOneDecimal(v float64) float64 {
	return math.Round(v*10) / 10
}

func recommendationTypeText(mode int) string {
	if mode == allPerfectMode {
		return "all_perfect"
	}
	return "full_combo"
}

func normalizeAchievementText(raw string) string {
	status := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case strings.Contains(status, "all_perfect"), strings.Contains(status, "all perfect"), status == "ap":
		return "all_perfect"
	case strings.Contains(status, "full_combo"), strings.Contains(status, "full combo"), status == "fc":
		return "full_combo"
	case strings.Contains(status, "clear"):
		return "clear"
	default:
		return "not_played"
	}
}

func (s *RandomService) loadUserAchievementRanks(ctx context.Context, userID uint) (map[uint]int, error) {
	items, err := s.recordRepo.ListUserRecordStatuses(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make(map[uint]int, len(items))
	for _, it := range items {
		rank := achievementRank(normalizeAchievementText(it.MusicAchievement))
		if current, ok := result[it.MusicDifficultyID]; !ok || rank > current {
			result[it.MusicDifficultyID] = rank
		}
	}
	return result, nil
}

func targetAchievementRank(mode int) int {
	if mode == allPerfectMode {
		return achievementRank("all_perfect")
	}
	return achievementRank("full_combo")
}

func filterDifficultiesByTargetRank(diffs []model.MusicDifficulty, userAchievementRanks map[uint]int, targetRank int) []model.MusicDifficulty {
	if len(diffs) == 0 {
		return nil
	}
	eligible := make([]model.MusicDifficulty, 0, len(diffs))
	for _, diff := range diffs {
		current := userAchievementRanks[diff.ID]
		if current < targetRank {
			eligible = append(eligible, diff)
		}
	}
	return eligible
}

func achievementRank(status string) int {
	switch status {
	case "all_perfect":
		return 3
	case "full_combo":
		return 2
	case "clear":
		return 1
	default:
		return 0
	}
}

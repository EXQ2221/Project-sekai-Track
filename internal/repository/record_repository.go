package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"Project_sekai_search/internal/dto"
	"Project_sekai_search/internal/model"

	"gorm.io/gorm"
)

type RecordRepository interface {
	GetMusicDifficultyByID(ctx context.Context, id uint) (*model.MusicDifficulty, error)
	GetMusicAchievementByID(ctx context.Context, id uint) (*model.MusicAchievement, error)
	UpsertUserRecordAndRefreshB30(ctx context.Context, userID uint, difficulty *model.MusicDifficulty, achievementID uint, scoreValue float64) (*model.Record, bool, float64, error)
	GetBest30ByUserID(ctx context.Context, userID uint, calcMode string) ([]dto.Best30Item, float64, error)
	GetB30TrendByUserID(ctx context.Context, userID uint, calcMode string) ([]dto.B30TrendPoint, error)
	UpsertB30TrendSnapshot(ctx context.Context, userID uint, bucketStart time.Time) error
	ListUserRecordStatuses(ctx context.Context, userID uint) ([]dto.UserRecordStatusItem, error)
	ListMusicAchievements(ctx context.Context) ([]model.MusicAchievement, error)
	GetDifficultyStatisticsOverview(ctx context.Context, userID uint, difficulty string) (*dto.RecordStatisticsBucket, error)
	ListDifficultyStatisticsByLevel(ctx context.Context, userID uint, difficulty string) ([]dto.RecordStatisticsBucket, error)
	ListGlobalStatisticsByLevelRange(ctx context.Context, userID uint, minLevel uint, maxLevel uint) ([]dto.RecordStatisticsBucket, error)
	DeleteUserRecordByDifficultyAndRefreshB30(ctx context.Context, userID uint, difficultyID uint) (bool, float64, error)
	FindRecordByDiffID(ctx context.Context, userID uint, diffsID uint) (model.Record, error)
}

type recordRepo struct {
	db *gorm.DB
}

func NewRecordRepo(db *gorm.DB) RecordRepository {
	return &recordRepo{db: db}
}

func (r *recordRepo) GetMusicDifficultyByID(ctx context.Context, id uint) (*model.MusicDifficulty, error) {
	var d model.MusicDifficulty
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&d).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *recordRepo) GetMusicAchievementByID(ctx context.Context, id uint) (*model.MusicAchievement, error) {
	var a model.MusicAchievement
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *recordRepo) UpsertUserRecordAndRefreshB30(ctx context.Context, userID uint, difficulty *model.MusicDifficulty, achievementID uint, scoreValue float64) (*model.Record, bool, float64, error) {
	var (
		rec     model.Record
		created bool
		avgB30  float64
	)

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		findErr := tx.Where("user_id = ? AND music_difficulty_id = ?", userID, difficulty.ID).First(&rec).Error
		if errors.Is(findErr, gorm.ErrRecordNotFound) {
			rec = model.Record{
				UserID:             userID,
				SongID:             difficulty.MusicID,
				MusicDifficultyID:  difficulty.ID,
				MusicAchievementID: achievementID,
				PlayLevel:          difficulty.PlayLevel,
				ScoreValue:         scoreValue,
			}
			if err := tx.Create(&rec).Error; err != nil {
				return err
			}
			created = true
		} else if findErr != nil {
			return findErr
		} else {
			rec.SongID = difficulty.MusicID
			rec.MusicAchievementID = achievementID
			rec.PlayLevel = difficulty.PlayLevel
			rec.ScoreValue = scoreValue
			if err := tx.Save(&rec).Error; err != nil {
				return err
			}
		}

		var err error
		avgB30, err = r.rebuildB30AndUpdateUserAvgTx(tx, userID)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, false, 0, err
	}

	return &rec, created, avgB30, nil
}

func (r *recordRepo) ListUserRecordStatuses(ctx context.Context, userID uint) ([]dto.UserRecordStatusItem, error) {
	type row struct {
		MusicDifficultyID uint
		MusicAchievement  string
		ScoreValue        float64
	}

	var rows []row
	err := r.db.WithContext(ctx).
		Table("records AS r").
		Select("r.music_difficulty_id AS music_difficulty_id, ma.status AS music_achievement, r.score_value AS score_value").
		Joins("LEFT JOIN music_achievements ma ON ma.id = r.music_achievement_id").
		Where("r.user_id = ?", userID).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	items := make([]dto.UserRecordStatusItem, 0, len(rows))
	for _, it := range rows {
		items = append(items, dto.UserRecordStatusItem{
			MusicDifficultyID: it.MusicDifficultyID,
			MusicAchievement:  it.MusicAchievement,
			ScoreValue:        it.ScoreValue,
		})
	}
	return items, nil
}

func (r *recordRepo) FindRecordByDiffID(ctx context.Context, userID uint, diffsID uint) (model.Record, error) {
	var record model.Record

	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("music_difficulty_id = ?", diffsID).
		First(&record).Error

	return record, err
}

func (r *recordRepo) ListMusicAchievements(ctx context.Context) ([]model.MusicAchievement, error) {
	var items []model.MusicAchievement
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *recordRepo) GetDifficultyStatisticsOverview(ctx context.Context, userID uint, difficulty string) (*dto.RecordStatisticsBucket, error) {
	type row struct {
		TotalCharts    int64
		NotPlayedCount int64
		ClearCount     int64
		FCCount        int64
		APCount        int64
	}

	diff := strings.ToLower(strings.TrimSpace(difficulty))
	var item row
	err := r.db.WithContext(ctx).
		Table("music_difficulties AS md").
		Select(
			"COUNT(md.id) AS total_charts, "+
				"COALESCE(SUM(CASE WHEN r.id IS NULL THEN 1 ELSE 0 END), 0) AS not_played_count, "+
				"COALESCE(SUM(CASE WHEN r.id IS NOT NULL AND (LOWER(ma.status) LIKE '%%clear%%') THEN 1 ELSE 0 END), 0) AS clear_count, "+
				"COALESCE(SUM(CASE WHEN r.id IS NOT NULL AND (LOWER(ma.status) LIKE '%%full_combo%%' OR LOWER(ma.status) LIKE '%%full combo%%' OR LOWER(ma.status) = 'fc') THEN 1 ELSE 0 END), 0) AS fc_count, "+
				"COALESCE(SUM(CASE WHEN r.id IS NOT NULL AND (LOWER(ma.status) LIKE '%%all_perfect%%' OR LOWER(ma.status) LIKE '%%all perfect%%' OR LOWER(ma.status) = 'ap') THEN 1 ELSE 0 END), 0) AS ap_count",
		).
		Joins("LEFT JOIN records r ON r.music_difficulty_id = md.id AND r.user_id = ?", userID).
		Joins("LEFT JOIN music_achievements ma ON ma.id = r.music_achievement_id").
		Where("LOWER(md.music_difficulty) = ?", diff).
		Scan(&item).Error
	if err != nil {
		return nil, err
	}

	bucket := dto.RecordStatisticsBucket{
		Label:          strings.ToUpper(diff),
		TotalCharts:    uint64(maxNonNegativeI64(item.TotalCharts)),
		NotPlayedCount: uint64(maxNonNegativeI64(item.NotPlayedCount)),
		ClearCount:     uint64(maxNonNegativeI64(item.ClearCount)),
		FCCount:        uint64(maxNonNegativeI64(item.FCCount)),
		APCount:        uint64(maxNonNegativeI64(item.APCount)),
	}
	fillStatisticsRates(&bucket)
	return &bucket, nil
}

func (r *recordRepo) ListDifficultyStatisticsByLevel(ctx context.Context, userID uint, difficulty string) ([]dto.RecordStatisticsBucket, error) {
	type row struct {
		PlayLevel      uint
		TotalCharts    int64
		NotPlayedCount int64
		ClearCount     int64
		FCCount        int64
		APCount        int64
	}

	diff := strings.ToLower(strings.TrimSpace(difficulty))
	var rows []row
	err := r.db.WithContext(ctx).
		Table("music_difficulties AS md").
		Select(
			"md.play_level AS play_level, "+
				"COUNT(md.id) AS total_charts, "+
				"COALESCE(SUM(CASE WHEN r.id IS NULL THEN 1 ELSE 0 END), 0) AS not_played_count, "+
				"COALESCE(SUM(CASE WHEN r.id IS NOT NULL AND (LOWER(ma.status) LIKE '%%clear%%') THEN 1 ELSE 0 END), 0) AS clear_count, "+
				"COALESCE(SUM(CASE WHEN r.id IS NOT NULL AND (LOWER(ma.status) LIKE '%%full_combo%%' OR LOWER(ma.status) LIKE '%%full combo%%' OR LOWER(ma.status) = 'fc') THEN 1 ELSE 0 END), 0) AS fc_count, "+
				"COALESCE(SUM(CASE WHEN r.id IS NOT NULL AND (LOWER(ma.status) LIKE '%%all_perfect%%' OR LOWER(ma.status) LIKE '%%all perfect%%' OR LOWER(ma.status) = 'ap') THEN 1 ELSE 0 END), 0) AS ap_count",
		).
		Joins("LEFT JOIN records r ON r.music_difficulty_id = md.id AND r.user_id = ?", userID).
		Joins("LEFT JOIN music_achievements ma ON ma.id = r.music_achievement_id").
		Where("LOWER(md.music_difficulty) = ?", diff).
		Group("md.play_level").
		Order("md.play_level ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	items := make([]dto.RecordStatisticsBucket, 0, len(rows))
	for _, it := range rows {
		bucket := dto.RecordStatisticsBucket{
			Label:          fmt.Sprintf("%s %d", strings.ToUpper(diff), it.PlayLevel),
			PlayLevel:      it.PlayLevel,
			TotalCharts:    uint64(maxNonNegativeI64(it.TotalCharts)),
			NotPlayedCount: uint64(maxNonNegativeI64(it.NotPlayedCount)),
			ClearCount:     uint64(maxNonNegativeI64(it.ClearCount)),
			FCCount:        uint64(maxNonNegativeI64(it.FCCount)),
			APCount:        uint64(maxNonNegativeI64(it.APCount)),
		}
		fillStatisticsRates(&bucket)
		items = append(items, bucket)
	}

	return items, nil
}

func (r *recordRepo) ListGlobalStatisticsByLevelRange(ctx context.Context, userID uint, minLevel uint, maxLevel uint) ([]dto.RecordStatisticsBucket, error) {
	type row struct {
		PlayLevel      uint
		TotalCharts    int64
		NotPlayedCount int64
		ClearCount     int64
		FCCount        int64
		APCount        int64
	}

	query := r.db.WithContext(ctx).
		Table("music_difficulties AS md").
		Select(
			"md.play_level AS play_level, "+
				"COUNT(md.id) AS total_charts, "+
				"COALESCE(SUM(CASE WHEN r.id IS NULL THEN 1 ELSE 0 END), 0) AS not_played_count, "+
				"COALESCE(SUM(CASE WHEN r.id IS NOT NULL AND (LOWER(ma.status) LIKE '%%clear%%') THEN 1 ELSE 0 END), 0) AS clear_count, "+
				"COALESCE(SUM(CASE WHEN r.id IS NOT NULL AND (LOWER(ma.status) LIKE '%%full_combo%%' OR LOWER(ma.status) LIKE '%%full combo%%' OR LOWER(ma.status) = 'fc') THEN 1 ELSE 0 END), 0) AS fc_count, "+
				"COALESCE(SUM(CASE WHEN r.id IS NOT NULL AND (LOWER(ma.status) LIKE '%%all_perfect%%' OR LOWER(ma.status) LIKE '%%all perfect%%' OR LOWER(ma.status) = 'ap') THEN 1 ELSE 0 END), 0) AS ap_count",
		).
		Joins("LEFT JOIN records r ON r.music_difficulty_id = md.id AND r.user_id = ?", userID).
		Joins("LEFT JOIN music_achievements ma ON ma.id = r.music_achievement_id")

	if minLevel > 0 {
		query = query.Where("md.play_level >= ?", minLevel)
	}
	if maxLevel > 0 {
		query = query.Where("md.play_level <= ?", maxLevel)
	}

	var rows []row
	if err := query.
		Group("md.play_level").
		Order("md.play_level ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]dto.RecordStatisticsBucket, 0, len(rows))
	for _, it := range rows {
		bucket := dto.RecordStatisticsBucket{
			Label:          fmt.Sprintf("Lv %d", it.PlayLevel),
			PlayLevel:      it.PlayLevel,
			TotalCharts:    uint64(maxNonNegativeI64(it.TotalCharts)),
			NotPlayedCount: uint64(maxNonNegativeI64(it.NotPlayedCount)),
			ClearCount:     uint64(maxNonNegativeI64(it.ClearCount)),
			FCCount:        uint64(maxNonNegativeI64(it.FCCount)),
			APCount:        uint64(maxNonNegativeI64(it.APCount)),
		}
		fillStatisticsRates(&bucket)
		items = append(items, bucket)
	}

	return items, nil
}

func (r *recordRepo) DeleteUserRecordByDifficultyAndRefreshB30(ctx context.Context, userID uint, difficultyID uint) (bool, float64, error) {
	var (
		deleted bool
		avgB30  float64
	)

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("user_id = ? AND music_difficulty_id = ?", userID, difficultyID).Delete(&model.Record{})
		if res.Error != nil {
			return res.Error
		}
		deleted = res.RowsAffected > 0

		var err error
		avgB30, err = r.rebuildB30AndUpdateUserAvgTx(tx, userID)
		return err
	})
	if err != nil {
		return false, 0, err
	}
	return deleted, avgB30, nil
}

func (r *recordRepo) GetBest30ByUserID(ctx context.Context, userID uint, calcMode string) ([]dto.Best30Item, float64, error) {
	type row struct {
		RecordID          uint
		ScoreValue        float64
		SongID            uint
		Title             string
		AssetBundleName   string
		MusicDifficultyID uint
		MusicDifficulty   string
		PlayLevel         uint
		ConstValue        float64 `gorm:"column:const_value"`
		MusicAchievement  string
	}

	baseLevelExpr := "r.play_level"
	if strings.EqualFold(strings.TrimSpace(calcMode), "const") {
		baseLevelExpr = "CASE WHEN md.`const` > 0 THEN md.`const` ELSE r.play_level END"
	}
	scoreExpr := fmt.Sprintf(
		"CASE "+
			"WHEN (LOWER(ma.status) LIKE '%%all_perfect%%' OR LOWER(ma.status) LIKE '%%all perfect%%' OR LOWER(ma.status) = 'ap') THEN (%s) "+
			"WHEN (LOWER(ma.status) LIKE '%%full_combo%%' OR LOWER(ma.status) LIKE '%%full combo%%' OR LOWER(ma.status) = 'fc') THEN "+
			"(CASE WHEN (%s) >= 33 THEN (%s) - 1.0 ELSE (%s) - 1.5 END) "+
			"ELSE (%s) - 5.0 END",
		baseLevelExpr, baseLevelExpr, baseLevelExpr, baseLevelExpr, baseLevelExpr,
	)

	var rows []row
	err := r.db.WithContext(ctx).
		Table("records AS r").
		Select(
			"r.id AS record_id, ("+scoreExpr+") AS score_value, "+
				"r.song_id AS song_id, m.title AS title, m.asset_bundle_name AS asset_bundle_name, "+
				"r.music_difficulty_id AS music_difficulty_id, md.music_difficulty AS music_difficulty, "+
				"r.play_level AS play_level, md.`const` AS const_value, ma.status AS music_achievement",
		).
		Joins("LEFT JOIN musics m ON m.id = r.song_id").
		Joins("LEFT JOIN music_difficulties md ON md.id = r.music_difficulty_id").
		Joins("LEFT JOIN music_achievements ma ON ma.id = r.music_achievement_id").
		Where("r.user_id = ?", userID).
		Order("score_value DESC").
		Order("r.id ASC").
		Limit(30).
		Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	items := make([]dto.Best30Item, 0, len(rows))
	var sum float64
	for _, it := range rows {
		sum += it.ScoreValue
		items = append(items, dto.Best30Item{
			RecordID:          it.RecordID,
			ScoreValue:        it.ScoreValue,
			SongID:            it.SongID,
			Title:             it.Title,
			AssetBundleName:   it.AssetBundleName,
			MusicDifficultyID: it.MusicDifficultyID,
			MusicDifficulty:   it.MusicDifficulty,
			PlayLevel:         it.PlayLevel,
			ConstValue:        it.ConstValue,
			MusicAchievement:  it.MusicAchievement,
		})
	}

	for i := range items {
		items[i].Rank = uint(i + 1)
	}

	avg := 0.0
	if len(items) > 0 {
		avg = sum / float64(len(items))
	}
	return items, avg, nil
}

func (r *recordRepo) GetB30TrendByUserID(ctx context.Context, userID uint, calcMode string) ([]dto.B30TrendPoint, error) {
	mode := "official"
	if strings.EqualFold(strings.TrimSpace(calcMode), "const") {
		mode = "const"
	}

	var rows []dto.B30TrendPoint
	if err := r.db.WithContext(ctx).
		Table("b30_trends").
		Select("avg_b30 AS avg_b30, created_at AS created_at").
		Where("user_id = ? AND calc_mode = ?", userID, mode).
		Order("created_at DESC").
		Order("id DESC").
		Limit(500).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	// Return chronological order for line chart rendering.
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	return rows, nil
}

func (r *recordRepo) rebuildB30AndUpdateUserAvgTx(tx *gorm.DB, userID uint) (float64, error) {
	var top []model.Record
	if err := tx.
		Where("user_id = ?", userID).
		Order("score_value DESC").
		Order("id ASC").
		Limit(30).
		Find(&top).Error; err != nil {
		return 0, err
	}

	if err := tx.Where("user_id = ?", userID).Delete(&model.B30Record{}).Error; err != nil {
		return 0, err
	}

	var avgB30 float64
	if len(top) > 0 {
		b30Rows := make([]model.B30Record, 0, len(top))
		var sum float64
		for i, item := range top {
			sum += item.ScoreValue
			b30Rows = append(b30Rows, model.B30Record{
				UserID:     userID,
				RecordID:   item.ID,
				Rank:       uint(i + 1),
				ScoreValue: item.ScoreValue,
			})
		}
		avgB30 = sum / float64(len(top))
		if err := tx.Create(&b30Rows).Error; err != nil {
			return 0, err
		}
	}

	constAvg, err := r.getB30AvgByModeTx(tx, userID, "const")
	if err != nil {
		return 0, err
	}
	if err := tx.Model(&model.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{
			"b30_avg":       avgB30,
			"b30_avg_const": constAvg,
		}).Error; err != nil {
		return 0, err
	}

	bucketStart := floorTo3Hour(time.Now())
	if err := r.upsertTrendPointTx(tx, userID, "official", bucketStart, avgB30); err != nil {
		return 0, err
	}
	if err := r.upsertTrendPointTx(tx, userID, "const", bucketStart, constAvg); err != nil {
		return 0, err
	}

	return avgB30, nil
}

func (r *recordRepo) UpsertB30TrendSnapshot(ctx context.Context, userID uint, bucketStart time.Time) error {
	bucketStart = bucketStart.In(time.Local).Truncate(time.Second)

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		officialAvg, err := r.getB30AvgByModeTx(tx, userID, "official")
		if err != nil {
			return err
		}
		constAvg, err := r.getB30AvgByModeTx(tx, userID, "const")
		if err != nil {
			return err
		}

		if err := r.upsertTrendPointTx(tx, userID, "official", bucketStart, officialAvg); err != nil {
			return err
		}
		if err := r.upsertTrendPointTx(tx, userID, "const", bucketStart, constAvg); err != nil {
			return err
		}
		return nil
	})
}

func (r *recordRepo) getB30AvgByModeTx(tx *gorm.DB, userID uint, calcMode string) (float64, error) {
	baseLevelExpr := "r.play_level"
	if strings.EqualFold(strings.TrimSpace(calcMode), "const") {
		baseLevelExpr = "CASE WHEN md.`const` > 0 THEN md.`const` ELSE r.play_level END"
	}
	scoreExpr := fmt.Sprintf(
		"CASE "+
			"WHEN (LOWER(ma.status) LIKE '%%all_perfect%%' OR LOWER(ma.status) LIKE '%%all perfect%%' OR LOWER(ma.status) = 'ap') THEN (%s) "+
			"WHEN (LOWER(ma.status) LIKE '%%full_combo%%' OR LOWER(ma.status) LIKE '%%full combo%%' OR LOWER(ma.status) = 'fc') THEN "+
			"(CASE WHEN (%s) >= 33 THEN (%s) - 1.0 ELSE (%s) - 1.5 END) "+
			"ELSE (%s) - 5.0 END",
		baseLevelExpr, baseLevelExpr, baseLevelExpr, baseLevelExpr, baseLevelExpr,
	)

	type avgRow struct {
		AvgB30 float64
	}
	var row avgRow
	err := tx.
		Table("(?) AS top30", tx.
			Table("records AS r").
			Select("("+scoreExpr+") AS score_value").
			Joins("LEFT JOIN music_difficulties md ON md.id = r.music_difficulty_id").
			Joins("LEFT JOIN music_achievements ma ON ma.id = r.music_achievement_id").
			Where("r.user_id = ?", userID).
			Order("score_value DESC").
			Order("r.id ASC").
			Limit(30),
		).
		Select("COALESCE(AVG(top30.score_value), 0) AS avg_b30").
		Scan(&row).Error
	if err != nil {
		return 0, err
	}
	return row.AvgB30, nil
}

func (r *recordRepo) upsertTrendPointTx(tx *gorm.DB, userID uint, calcMode string, bucketStart time.Time, avg float64) error {
	var last model.B30Trend
	err := tx.
		Where("user_id = ? AND calc_mode = ? AND created_at = ?", userID, calcMode, bucketStart).
		First(&last).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		item := model.B30Trend{
			UserID:    userID,
			CalcMode:  calcMode,
			AvgB30:    avg,
			CreatedAt: bucketStart,
		}
		return tx.Create(&item).Error
	case err != nil:
		return err
	default:
		return tx.Model(&model.B30Trend{}).
			Where("id = ?", last.ID).
			Update("avg_b30", avg).Error
	}
}

func floorTo3Hour(t time.Time) time.Time {
	local := t.In(time.Local)
	return time.Date(local.Year(), local.Month(), local.Day(), (local.Hour()/3)*3, 0, 0, 0, local.Location()).Truncate(time.Second)
}

func fillStatisticsRates(bucket *dto.RecordStatisticsBucket) {
	if bucket == nil || bucket.TotalCharts == 0 {
		return
	}
	base := float64(bucket.TotalCharts)
	bucket.NotPlayedRate = float64(bucket.NotPlayedCount) * 100.0 / base
	bucket.ClearRate = float64(bucket.ClearCount) * 100.0 / base
	bucket.FCRate = float64(bucket.FCCount) * 100.0 / base
	bucket.APRate = float64(bucket.APCount) * 100.0 / base
}

func maxNonNegativeI64(v int64) int64 {
	if v < 0 {
		return 0
	}
	return v
}

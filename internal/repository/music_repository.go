package repository

import (
	"context"
	"strings"

	"Project_sekai_search/internal/dto"
	"Project_sekai_search/internal/model"

	"gorm.io/gorm"
)

type MusicRepository interface {
	ListMusics(ctx context.Context, q dto.ListMusicQuery) ([]model.Music, int64, error)
	GetMusicByID(ctx context.Context, id uint) (*model.Music, error)
	UpdateMusicAlias(ctx context.Context, id uint, alias string) error
	ListDifficultyStatsByMusicID(ctx context.Context, musicID uint) ([]dto.MusicDifficultyStat, error)
	FindDifficultiesByLevel(ctx context.Context, targetConst uint) ([]model.MusicDifficulty, error)
	FindDifficultiesByConst(ctx context.Context, targetConst float64) ([]model.MusicDifficulty, error)
	FindMusicByDifficultyID(ctx context.Context, targetID uint) (uint, error)
}

type musicRepo struct {
	db *gorm.DB
}

func NewMusicRepo(db *gorm.DB) MusicRepository {
	return &musicRepo{db: db}
}

func (r *musicRepo) ListMusics(ctx context.Context, q dto.ListMusicQuery) ([]model.Music, int64, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.Size <= 0 {
		q.Size = 20
	}
	if q.Size > 100 {
		q.Size = 100
	}
	offset := (q.Page - 1) * q.Size

	base := r.db.WithContext(ctx).Model(&model.Music{})
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		base = base.Where("title LIKE ? OR alias LIKE ?", like, like)
	}
	if len(q.Filters) > 0 {
		conds := make([]string, 0, len(q.Filters))
		args := make([]any, 0, len(q.Filters)*2)
		for _, f := range q.Filters {
			if strings.TrimSpace(f.Difficulty) == "" {
				conds = append(conds, "(md.play_level = ?)")
				args = append(args, f.PlayLevel)
			} else {
				conds = append(conds, "(md.music_difficulty = ? AND md.play_level = ?)")
				args = append(args, f.Difficulty, f.PlayLevel)
			}
		}
		existsSQL := "EXISTS (SELECT 1 FROM music_difficulties md WHERE md.music_id = musics.id AND (" + strings.Join(conds, " OR ") + "))"
		base = base.Where(existsSQL, args...)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	orderBy := "musics.id DESC"
	if !q.SortDescByID {
		orderBy = "musics.id ASC"
	}

	var musics []model.Music
	if err := base.
		Preload("Difficulties").
		Order(orderBy).
		Offset(offset).
		Limit(q.Size).
		Find(&musics).Error; err != nil {
		return nil, 0, err
	}

	return musics, total, nil
}

func (r *musicRepo) GetMusicByID(ctx context.Context, id uint) (*model.Music, error) {
	var music model.Music
	if err := r.db.WithContext(ctx).
		Preload("Difficulties").
		Where("id = ?", id).
		First(&music).Error; err != nil {
		return nil, err
	}

	return &music, nil
}

func (r *musicRepo) UpdateMusicAlias(ctx context.Context, id uint, alias string) error {
	return r.db.WithContext(ctx).
		Model(&model.Music{}).
		Where("id = ?", id).
		Update("alias", alias).Error
}

func (r *musicRepo) ListDifficultyStatsByMusicID(ctx context.Context, musicID uint) ([]dto.MusicDifficultyStat, error) {
	type row struct {
		MusicDifficultyID uint
		MusicDifficulty   string
		PlayLevel         uint
		ConstValue        float64
		TotalNoteCount    uint
		PlayedCount       int64
		FCCount           int64
		APCount           int64
	}

	var rows []row
	err := r.db.WithContext(ctx).
		Table("music_difficulties AS md").
		Select(
			"md.id AS music_difficulty_id, md.music_difficulty AS music_difficulty, md.play_level AS play_level, md.`const` AS const_value, md.total_note_count AS total_note_count, "+
				"COUNT(r.id) AS played_count, "+
				"COALESCE(SUM(CASE "+
				"WHEN (LOWER(ma.status) LIKE '%%full_combo%%' OR LOWER(ma.status) LIKE '%%full combo%%' OR LOWER(ma.status) = 'fc') THEN 1 "+
				"ELSE 0 END), 0) AS fc_count, "+
				"COALESCE(SUM(CASE "+
				"WHEN (LOWER(ma.status) LIKE '%%all_perfect%%' OR LOWER(ma.status) LIKE '%%all perfect%%' OR LOWER(ma.status) = 'ap') THEN 1 "+
				"ELSE 0 END), 0) AS ap_count",
		).
		Joins("LEFT JOIN records r ON r.music_difficulty_id = md.id").
		Joins("LEFT JOIN music_achievements ma ON ma.id = r.music_achievement_id").
		Where("md.music_id = ?", musicID).
		Group("md.id, md.music_difficulty, md.play_level, md.`const`, md.total_note_count").
		Order("CASE LOWER(md.music_difficulty) WHEN 'easy' THEN 1 WHEN 'normal' THEN 2 WHEN 'hard' THEN 3 WHEN 'expert' THEN 4 WHEN 'master' THEN 5 WHEN 'append' THEN 6 ELSE 99 END ASC").
		Order("md.play_level ASC").
		Order("md.id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	stats := make([]dto.MusicDifficultyStat, 0, len(rows))
	for _, it := range rows {
		playedCount := uint64(maxI64(it.PlayedCount, 0))
		fcCount := uint64(maxI64(it.FCCount, 0))
		apCount := uint64(maxI64(it.APCount, 0))

		fcRate := 0.0
		apRate := 0.0
		if playedCount > 0 {
			base := float64(playedCount)
			fcRate = float64(fcCount) * 100.0 / base
			apRate = float64(apCount) * 100.0 / base
		}

		stats = append(stats, dto.MusicDifficultyStat{
			MusicDifficultyID: it.MusicDifficultyID,
			MusicDifficulty:   it.MusicDifficulty,
			PlayLevel:         it.PlayLevel,
			ConstValue:        it.ConstValue,
			TotalNoteCount:    it.TotalNoteCount,
			PlayedCount:       playedCount,
			FCCount:           fcCount,
			APCount:           apCount,
			FCRate:            fcRate,
			APRate:            apRate,
		})
	}
	return stats, nil
}

func (r *musicRepo) FindMusicByDifficultyID(ctx context.Context, targetID uint) (uint, error) {
	var musicID uint
	err := r.db.WithContext(ctx).
		Model(&model.MusicDifficulty{}).
		Select("music_id").
		Where("id = ?", targetID).
		Scan(&musicID).Error
	return musicID, err
}

func (r *musicRepo) FindDifficultiesByLevel(ctx context.Context, targetConst uint) ([]model.MusicDifficulty, error) {
	var musicDifficulties []model.MusicDifficulty
	err := r.db.WithContext(ctx).
		Model(&model.MusicDifficulty{}).
		Where("play_level = ?", targetConst).
		Find(&musicDifficulties).Error

	return musicDifficulties, err

}

func (r *musicRepo) FindDifficultiesByConst(ctx context.Context, targetConst float64) ([]model.MusicDifficulty, error) {
	var musicDifficulties []model.MusicDifficulty
	err := r.db.WithContext(ctx).
		Model(&model.MusicDifficulty{}).
		Where("const = ?", targetConst).
		Find(&musicDifficulties).Error

	return musicDifficulties, err

}

func maxI64(v int64, floor int64) int64 {
	if v < floor {
		return floor
	}
	return v
}

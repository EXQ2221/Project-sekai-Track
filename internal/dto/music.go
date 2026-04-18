package dto

import "Project_sekai_search/internal/model"

type ListMusicQuery struct {
	Page             int                     `form:"page"`
	Size             int                     `form:"size"`
	Keyword          string                  `form:"keyword"`
	DifficultyLevels string                  `form:"difficulty_levels"`
	Sort             string                  `form:"sort"`
	Filters          []DifficultyLevelFilter `form:"-"`
	SortDescByID     bool                    `form:"-"`
}

type DifficultyLevelFilter struct {
	Difficulty string `json:"difficulty,omitempty"`
	PlayLevel  uint   `json:"play_level"`
}

type MusicDifficultyStat struct {
	MusicDifficultyID uint    `json:"music_difficulty_id"`
	MusicDifficulty   string  `json:"music_difficulty"`
	PlayLevel         uint    `json:"play_level"`
	ConstValue        float64 `json:"const_value"`
	TotalNoteCount    uint    `json:"total_note_count"`
	PlayedCount       uint64  `json:"played_count"`
	FCCount           uint64  `json:"fc_count"`
	APCount           uint64  `json:"ap_count"`
	FCRate            float64 `json:"fc_rate"`
	APRate            float64 `json:"ap_rate"`
}

type MusicDetailResponse struct {
	Music           *model.Music          `json:"music"`
	DifficultyStats []MusicDifficultyStat `json:"difficulty_stats"`
	TotalCount      uint64                `json:"total_count"`
	TotalNoteCount  uint                  `json:"total_note_count"`
	FCTotalCount    uint64                `json:"fc_total_count"`
	APTotalCount    uint64                `json:"ap_total_count"`
}

type AddMusicAliasRequest struct {
	Alias string `json:"alias" binding:"required"`
}

package dto

import "time"

type Best30Item struct {
	Rank              uint    `json:"rank"`
	RecordID          uint    `json:"record_id"`
	ScoreValue        float64 `json:"score_value"`
	SongID            uint    `json:"song_id"`
	Title             string  `json:"title"`
	AssetBundleName   string  `json:"assetbundleName"`
	MusicDifficultyID uint    `json:"music_difficulty_id"`
	MusicDifficulty   string  `json:"music_difficulty"`
	PlayLevel         uint    `json:"play_level"`
	ConstValue        float64 `json:"const_value"`
	MusicAchievement  string  `json:"music_achievement"`
}

type B30TrendPoint struct {
	AvgB30    float64   `json:"avg_b30"`
	CreatedAt time.Time `json:"created_at"`
}

package dto

type RandomMusicRecommendationQuery struct {
	CalcMode string
}

type RandomResponse struct {
	SongID            uint    `json:"song_id"`
	AssetBundleName   string  `json:"assetbundleName"`
	Title             string  `json:"title"`
	MusicDifficultyID uint    `json:"music_difficulty_id"`
	MusicDifficulty   string  `json:"music_difficulty"`
	PlayLevel         uint    `json:"play_level"`
	ConstValue        float64 `json:"const"`
	CalcMode          string  `json:"calc_mode"`
	Type              string  `json:"type"`
	UserAchievementID uint    `json:"user_achievement_id"`
	UserAchievement   string  `json:"user_achievement"`
	UserScoreValue    float64 `json:"user_score_value"`
}

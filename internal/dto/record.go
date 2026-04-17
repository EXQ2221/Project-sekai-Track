package dto

type UploadRecordRequest struct {
	MusicDifficultyID  uint `json:"music_difficulty_id" binding:"required"`
	MusicAchievementID uint `json:"music_achievement_id" binding:"required"`
}

type DeleteRecordRequest struct {
	MusicDifficultyID uint `json:"music_difficulty_id" binding:"required"`
}

type UserRecordStatusItem struct {
	MusicDifficultyID uint    `json:"music_difficulty_id"`
	MusicAchievement  string  `json:"music_achievement"`
	ScoreValue        float64 `json:"score_value"`
}

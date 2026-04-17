package dto

type MyProfileResponse struct {
	ID                uint    `json:"id"`
	Username          string  `json:"username"`
	AvatarURL         string  `json:"avatar_url"`
	Profile           string  `json:"profile"`
	Character         string  `json:"character"`
	CharacterName     string  `json:"character_name"`
	CharacterImageURL string  `json:"character_image_url"`
	B30Avg            float64 `json:"b30_avg"`
}

type UpdateProfileRequest struct {
	Profile string `json:"profile" binding:"max=255"`
}

type UpdateCharacterRequest struct {
	Character string `json:"character" binding:"max=255"`
}

type CharacterOption struct {
	Key      string `json:"key"`
	Name     string `json:"name"`
	ImageURL string `json:"image_url"`
}

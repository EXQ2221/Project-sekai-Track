package dto

type RecordStatisticsBucket struct {
	Label          string  `json:"label"`
	PlayLevel      uint    `json:"play_level"`
	TotalCharts    uint64  `json:"total_charts"`
	NotPlayedCount uint64  `json:"not_played_count"`
	ClearCount     uint64  `json:"clear_count"`
	FCCount        uint64  `json:"fc_count"`
	APCount        uint64  `json:"ap_count"`
	NotPlayedRate  float64 `json:"not_played_rate"`
	ClearRate      float64 `json:"clear_rate"`
	FCRate         float64 `json:"fc_rate"`
	APRate         float64 `json:"ap_rate"`
}

type RecordStatisticsResponse struct {
	Difficulty  string                   `json:"difficulty"`
	Mode        string                   `json:"mode"`
	MinLevel    uint                     `json:"min_level"`
	MaxLevel    uint                     `json:"max_level"`
	TotalCharts uint64                   `json:"total_charts"`
	Buckets     []RecordStatisticsBucket `json:"buckets"`
}

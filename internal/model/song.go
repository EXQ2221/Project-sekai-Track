package model

import "time"

type Music struct {
	ID              uint              `gorm:"primaryKey;autoIncrement:false" json:"id"` // 瀹樻柟ID锛屼笉鑷
	Title           string            `gorm:"type:varchar(255);not null" json:"title"`
	Alias           string            `gorm:"type:varchar(255)" json:"alias"` // 鍒悕鍙互涓虹┖
	Composer        string            `gorm:"type:varchar(255)" json:"composer"`
	Difficulties    []MusicDifficulty `gorm:"foreignKey:MusicID" json:"difficulties,omitempty"`
	CreatorArtistID uint              `gorm:"index" json:"creatorArtistId"`
	AssetBundleName string            `gorm:"type:varchar(255);not null" json:"assetbundleName"`
	CoverURL        string            `gorm:"-" json:"cover_url"`
}

func (Music) TableName() string { return "musics" }

type MusicDifficulty struct {
	ID              uint    `gorm:"primaryKey;autoIncrement:false" json:"id"`
	MusicID         uint    `gorm:"index;not null" json:"musicId"`                    // 绱㈠紩鎻愬崌鍏宠仈鏌ヨ鏁堢巼
	MusicDifficulty string  `gorm:"type:varchar(20);not null" json:"musicDifficulty"` // master, expert...
	PlayLevel       uint    `gorm:"not null" json:"playLevel"`
	Const           float64 `gorm:"column:const;type:decimal(4,1);not null;default:0" json:"const"`
	TotalNoteCount  uint    `gorm:"not null" json:"totalNoteCount"`
}

func (MusicDifficulty) TableName() string { return "music_difficulties" }

type MusicAchievement struct {
	ID                  uint   `gorm:"primaryKey;autoIncrement:false" json:"id"`
	MusicDifficultyType string `gorm:"type:varchar(20);not null" json:"musicDifficultyType"`
	Status              string `gorm:"type:varchar(50);not null" json:"musicAchievementType"`
}

func (MusicAchievement) TableName() string { return "music_achievements" }

type Record struct {
	ID                 uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID             uint      `gorm:"index:idx_user_record;not null" json:"user_id"`                              // 鐢ㄦ埛缁村害绱㈠紩
	SongID             uint      `gorm:"index:idx_user_record;not null" json:"song_id"`                              // 姝屾洸缁村害绱㈠紩
	MusicDifficultyID  uint      `gorm:"index;not null" json:"music_difficulty_id"`                                  // 鏂逛究鐩存帴杩為毦搴﹁〃
	MusicAchievementID uint      `gorm:"not null" json:"music_achievement_id"`                                       // 鍏宠仈瀛楀吀琛?
	PlayLevel          uint      `gorm:"index;not null" json:"play_level"`                                           // 鍐椾綑瀛楁锛屽姞绱㈠紩鏂逛究鎸夌瓑绾х瓫閫?
	ScoreValue         float64   `gorm:"column:score_value;type:decimal(6,2);not null;default:0" json:"score_value"` // b30璁＄畻鐢ㄦ垚缁╁€?
	CreatedAt          time.Time `gorm:"autoCreateTime" json:"created_at"`                                           // 鑷姩璁板綍鍒涘缓鏃堕棿
	UpdatedAt          time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Record) TableName() string { return "records" }

type B30Record struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     uint      `gorm:"index;not null" json:"user_id"`
	RecordID   uint      `gorm:"not null" json:"record_id"`
	Rank       uint      `gorm:"not null" json:"rank"`
	ScoreValue float64   `gorm:"column:score_value;type:decimal(6,2);not null" json:"score_value"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (B30Record) TableName() string { return "b30_records" }

type B30Trend struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint      `gorm:"index:idx_b30trend_user_mode_time,priority:1;not null" json:"user_id"`
	CalcMode  string    `gorm:"type:varchar(16);index:idx_b30trend_user_mode_time,priority:2;not null" json:"calc_mode"`
	AvgB30    float64   `gorm:"column:avg_b30;type:decimal(7,4);not null;default:0" json:"avg_b30"`
	CreatedAt time.Time `gorm:"index:idx_b30trend_user_mode_time,priority:3;autoCreateTime" json:"created_at"`
}

func (B30Trend) TableName() string { return "b30_trends" }

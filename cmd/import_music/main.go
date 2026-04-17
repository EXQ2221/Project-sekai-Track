package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"Project_sekai_search/internal/config"
	"Project_sekai_search/internal/model"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type musicJSON struct {
	ID              uint   `json:"id"`
	Title           string `json:"title"`
	Composer        string `json:"composer"`
	CreatorArtistID uint   `json:"creatorArtistId"`
	AssetBundleName string `json:"assetbundleName"`
}

type musicDifficultyJSON struct {
	ID              uint    `json:"id"`
	MusicID         uint    `json:"musicId"`
	MusicDifficulty string  `json:"musicDifficulty"`
	PlayLevel       uint    `json:"playLevel"`
	Const           float64 `json:"const"`
	TotalNoteCount  uint    `json:"totalNoteCount"`
}

func readJSON[T any](path string) ([]T, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Handle UTF-8 BOM to avoid JSON parse errors like:
	// invalid character 'ï' looking for beginning of value.
	b = bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF})
	var items []T
	if err := json.Unmarshal(b, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func main() {
	musicPath := flag.String("musics", "musics.json", "path to musics.json")
	diffPath := flag.String("difficulties", "musicDifficulties.json", "path to musicDifficulties.json")
	flag.Parse()

	_ = godotenv.Overload(".env.local")
	_ = godotenv.Load(".env")
	config.InitMySQL()

	musicsRaw, err := readJSON[musicJSON](*musicPath)
	if err != nil {
		panic(fmt.Errorf("read musics json failed: %w", err))
	}
	diffsRaw, err := readJSON[musicDifficultyJSON](*diffPath)
	if err != nil {
		panic(fmt.Errorf("read music difficulties json failed: %w", err))
	}

	musics := make([]model.Music, 0, len(musicsRaw))
	for _, m := range musicsRaw {
		musics = append(musics, model.Music{
			ID:              m.ID,
			Title:           m.Title,
			Composer:        m.Composer,
			CreatorArtistID: m.CreatorArtistID,
			AssetBundleName: m.AssetBundleName,
		})
	}

	difficulties := make([]model.MusicDifficulty, 0, len(diffsRaw))
	for _, d := range diffsRaw {
		difficulties = append(difficulties, model.MusicDifficulty{
			ID:              d.ID,
			MusicID:         d.MusicID,
			MusicDifficulty: d.MusicDifficulty,
			PlayLevel:       d.PlayLevel,
			Const:           d.Const,
			TotalNoteCount:  d.TotalNoteCount,
		})
	}

	diffs := []string{"easy", "normal", "hard", "expert", "master", "append"}
	statuses := []string{"clear", "full_combo", "all_perfect"}
	achievements := make([]model.MusicAchievement, 0, len(diffs)*len(statuses))
	var achievementID uint = 1
	for _, d := range diffs {
		for _, s := range statuses {
			achievements = append(achievements, model.MusicAchievement{
				ID:                  achievementID,
				MusicDifficultyType: d,
				Status:              s,
			})
			achievementID++
		}
	}

	err = config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.AutoMigrate(&model.Music{}, &model.MusicDifficulty{}, &model.MusicAchievement{}); err != nil {
			return err
		}

		if len(musics) > 0 {
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"title", "composer", "creator_artist_id", "asset_bundle_name",
				}),
			}).Create(&musics).Error; err != nil {
				return err
			}
		}

		if len(difficulties) > 0 {
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"music_id", "music_difficulty", "play_level", "const", "total_note_count",
				}),
			}).Create(&difficulties).Error; err != nil {
				return err
			}
		}

		if len(achievements) > 0 {
			if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.MusicAchievement{}).Error; err != nil {
				return err
			}
			if err := tx.Create(&achievements).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		panic(fmt.Errorf("import transaction failed: %w", err))
	}

	fmt.Printf("import done: musics=%d, music_difficulties=%d, music_achievements=%d\n", len(musics), len(difficulties), len(achievements))
	fmt.Printf("source files: %s, %s\n", filepath.Clean(*musicPath), filepath.Clean(*diffPath))
}

package main

import (
	"Project_sekai_search/internal/config"
	"Project_sekai_search/internal/model"
	"Project_sekai_search/internal/repository"
	"Project_sekai_search/internal/router"
	"Project_sekai_search/internal/service"
	"log"
	"strings"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

func main() {

	_ = godotenv.Load(".env")
	config.InitMySQL()
	config.InitRedis()
	if err := config.DB.AutoMigrate(
		&model.User{},
		&model.Session{},
		&model.RefreshToken{},
		&model.SecurityEvent{},
		&model.Music{},
		&model.MusicDifficulty{},
		&model.MusicAchievement{},
		&model.Record{},
		&model.B30Record{},
		&model.B30Trend{},
	); err != nil {
		log.Fatal("auto migrate failed: ", err)
	}
	if err := ensureLegacyRecordSchemaCompatible(); err != nil {
		log.Fatal("legacy records schema fix failed: ", err)
	}
	if err := reseedMusicAchievements(); err != nil {
		log.Fatal("seed music achievements failed: ", err)
	}

	userRepo := repository.NewUserRepo(config.DB)
	musicRepo := repository.NewMusicRepo(config.DB)
	recordRepo := repository.NewRecordRepo(config.DB)
	sessionRepo := repository.NewSessionRepo(config.DB)
	refreshRepo := repository.NewRefreshTokenRepo(config.DB)
	securityEventRepo := repository.NewSecurityEventRepo(config.DB)

	userService := service.NewUserService(userRepo)
	musicService := service.NewMusicService(musicRepo)
	recordService := service.NewRecordService(recordRepo)
	authService := service.NewAuthService(userRepo, sessionRepo, refreshRepo, securityEventRepo, config.DB)
	randomService := service.NewRandomService(userRepo, musicRepo, recordRepo)
	userService.SetAuthService(authService)

	router.InitRouter(authService, userService, musicService, recordService, randomService)
}

func ensureLegacyRecordSchemaCompatible() error {
	// 先确保主键ID是自增，满足 record.id 从 1 开始递增
	if err := config.DB.Exec("ALTER TABLE records MODIFY COLUMN id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT").Error; err != nil {
		return err
	}

	// 兼容旧表结构：旧版 records 里可能残留 chart_id/grade
	// 先把 chart_id 改为可空（若仍存在），避免插入新记录时报“no default value”
	if err := execIgnoreKnown(config.DB, "ALTER TABLE records MODIFY COLUMN chart_id BIGINT UNSIGNED NULL DEFAULT NULL"); err != nil {
		return err
	}

	// 再尝试清理旧约束和旧字段（不存在则忽略）
	stmts := []string{
		"ALTER TABLE records DROP FOREIGN KEY fk_records_chart",
		"ALTER TABLE records DROP INDEX idx_records_chart",
		"ALTER TABLE records DROP INDEX uk_records_user_chart",
		"ALTER TABLE records DROP COLUMN chart_id",
		"ALTER TABLE records DROP COLUMN grade",
	}
	for _, stmt := range stmts {
		if err := execIgnoreKnown(config.DB, stmt); err != nil {
			return err
		}
	}
	return nil
}

func execIgnoreKnown(db *gorm.DB, stmt string) error {
	err := db.Exec(stmt).Error
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "doesn't exist") ||
		strings.Contains(msg, "can't drop") ||
		strings.Contains(msg, "check that column/key exists") ||
		strings.Contains(msg, "unknown column") ||
		strings.Contains(msg, "duplicate key name") {
		return nil
	}
	return err
}

func reseedMusicAchievements() error {
	diffs := []string{"easy", "normal", "hard", "expert", "master", "append"}
	statuses := []string{"clear", "full_combo", "all_perfect"}

	seedAchievements := make([]model.MusicAchievement, 0, len(diffs)*len(statuses))
	var id uint = 1
	for _, d := range diffs {
		for _, s := range statuses {
			seedAchievements = append(seedAchievements, model.MusicAchievement{
				ID:                  id,
				MusicDifficultyType: d,
				Status:              s,
			})
			id++
		}
	}

	return config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.MusicAchievement{}).Error; err != nil {
			return err
		}
		return tx.Create(&seedAchievements).Error
	})
}

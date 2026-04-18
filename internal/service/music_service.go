package service

import (
	"context"
	"errors"
	"path"
	"strconv"
	"strings"
	"unicode"

	"Project_sekai_search/internal/dto"
	"Project_sekai_search/internal/model"
	"Project_sekai_search/internal/pkg/errcode"
	"Project_sekai_search/internal/repository"
	"gorm.io/gorm"
)

type MusicService struct {
	musicRepo repository.MusicRepository
}

func NewMusicService(musicRepo repository.MusicRepository) *MusicService {
	return &MusicService{musicRepo: musicRepo}
}

func (s *MusicService) ListMusics(ctx context.Context, q dto.ListMusicQuery) ([]model.Music, int64, int, int, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.Size <= 0 {
		q.Size = 20
	}
	if q.Size > 100 {
		q.Size = 100
	}
	q.SortDescByID = parseSortByID(q.Sort)

	filters, err := parseDifficultyLevels(q.DifficultyLevels)
	if err != nil {
		return nil, 0, 0, 0, errcode.ErrBadRequest
	}
	q.Filters = filters

	list, total, err := s.musicRepo.ListMusics(ctx, q)
	if err != nil {
		return nil, 0, 0, 0, errcode.ErrInternal
	}
	enrichMusicCoverURL(list)

	return list, total, q.Page, q.Size, nil
}

func (s *MusicService) GetMusicDetail(ctx context.Context, id uint) (*dto.MusicDetailResponse, error) {
	music, err := s.musicRepo.GetMusicByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrNotFound
		}
		return nil, errcode.ErrInternal
	}

	stats, err := s.musicRepo.ListDifficultyStatsByMusicID(ctx, id)
	if err != nil {
		return nil, errcode.ErrInternal
	}

	music.CoverURL = buildCoverURL(music.AssetBundleName)

	resp := &dto.MusicDetailResponse{
		Music:           music,
		DifficultyStats: stats,
	}

	for _, it := range stats {
		resp.TotalCount += it.PlayedCount
		resp.TotalNoteCount += it.TotalNoteCount
		resp.FCTotalCount += it.FCCount
		resp.APTotalCount += it.APCount
	}

	return resp, nil
}

func (s *MusicService) AddMusicAlias(ctx context.Context, id uint, alias string) (string, error) {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return "", errcode.ErrBadRequest
	}

	music, err := s.musicRepo.GetMusicByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errcode.ErrNotFound
		}
		return "", errcode.ErrInternal
	}

	merged := mergeMusicAlias(music.Alias, alias)
	if merged == music.Alias {
		return merged, nil
	}

	if err := s.musicRepo.UpdateMusicAlias(ctx, id, merged); err != nil {
		return "", errcode.ErrInternal
	}
	return merged, nil
}

func parseSortByID(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "oldest", "asc", "id_asc":
		return false
	default:
		return true
	}
}

func parseDifficultyLevels(raw string) ([]dto.DifficultyLevelFilter, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	filters := make([]dto.DifficultyLevelFilter, 0, len(parts))
	for _, p := range parts {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" {
			continue
		}

		if allDigits(p) {
			level, err := strconv.Atoi(p)
			if err != nil || level <= 0 {
				return nil, errcode.ErrBadRequest
			}
			filters = append(filters, dto.DifficultyLevelFilter{
				PlayLevel: uint(level),
			})
			continue
		}

		i := 0
		for i < len(p) && unicode.IsLetter(rune(p[i])) {
			i++
		}
		if i == 0 || i == len(p) {
			return nil, errcode.ErrBadRequest
		}

		diff := p[:i]
		if diff == "matser" {
			diff = "master"
		}
		switch diff {
		case "easy", "normal", "hard", "expert", "master", "append":
		default:
			return nil, errcode.ErrBadRequest
		}

		level, err := strconv.Atoi(p[i:])
		if err != nil || level <= 0 {
			return nil, errcode.ErrBadRequest
		}

		filters = append(filters, dto.DifficultyLevelFilter{
			Difficulty: diff,
			PlayLevel:  uint(level),
		})
	}

	return filters, nil
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func enrichMusicCoverURL(list []model.Music) {
	for i := range list {
		list[i].CoverURL = buildCoverURL(list[i].AssetBundleName)
	}
}

func buildCoverURL(assetBundleName string) string {
	name := strings.TrimSpace(assetBundleName)
	if name == "" {
		return ""
	}
	return path.Join("/static/assets", name+".png")
}

func mergeMusicAlias(raw, added string) string {
	items := splitAlias(raw)
	seen := make(map[string]struct{}, len(items)+1)
	out := make([]string, 0, len(items)+1)

	for _, it := range items {
		key := strings.ToLower(strings.TrimSpace(it))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, it)
	}

	addedKey := strings.ToLower(strings.TrimSpace(added))
	if addedKey != "" {
		if _, ok := seen[addedKey]; !ok {
			out = append(out, added)
		}
	}

	return strings.Join(out, " / ")
}

func splitAlias(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "-" {
		return nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		switch r {
		case '/', '|', ',', '，', '、', ';', '；':
			return true
		default:
			return false
		}
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" && p != "-" {
			out = append(out, p)
		}
	}
	return out
}

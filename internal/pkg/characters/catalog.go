package characters

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Item struct {
	Key      string
	Name     string
	ImageURL string
}

func List(baseDir string) ([]Item, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Item{}, nil
		}
		return nil, err
	}

	items := make([]Item, 0, 16)

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ext := strings.ToLower(filepath.Ext(name))
		base := strings.TrimSpace(strings.TrimSuffix(name, ext))
		if base == "" {
			continue
		}
		switch ext {
		case ".png", ".jpg", ".jpeg", ".webp":
			items = append(items, Item{
				Key:      base,
				Name:     displayNameFromBase(base),
				ImageURL: toStaticURL(filepath.Join("static", "characters", name)),
			})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	return items, nil
}

func FindByKey(baseDir, key string) (Item, bool, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return Item{}, false, nil
	}
	items, err := List(baseDir)
	if err != nil {
		return Item{}, false, err
	}
	for _, item := range items {
		if item.Key == key {
			return item, true, nil
		}
	}
	return Item{}, false, nil
}

func displayNameFromBase(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		return "Unknown"
	}
	base = strings.ReplaceAll(base, "_", " ")
	base = strings.ReplaceAll(base, "-", " ")
	base = strings.TrimSpace(base)
	if base == "" {
		return "Unknown"
	}
	return base
}

func toStaticURL(localPath string) string {
	parts := strings.Split(filepath.ToSlash(localPath), "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return "/" + strings.Join(parts, "/")
}

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	sourceJSON   = "musics.json"
	outDir       = "static/assets"
	baseImageURL = "https://storage.sekai.best/sekai-jp-assets/music/jacket/%s/%s.png"
)

type musicRow struct {
	ID              int    `json:"id"`
	AssetBundleName string `json:"assetbundleName"`
}

type coverMapRow struct {
	ID              int    `json:"id"`
	AssetBundleName string `json:"assetbundle_name"`
	URL             string `json:"url"`
	File            string `json:"file"`
}

func main() {
	startID := flag.Int("start-id", 130, "only download covers with music id >= this value")
	flag.Parse()

	rows, err := loadMusicRows(sourceJSON)
	if err != nil {
		panic(err)
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		panic(err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	mapping := make([]coverMapRow, 0, len(rows))
	okCount := 0

	for _, row := range rows {
		if row.ID < *startID {
			continue
		}

		name := strings.TrimSpace(row.AssetBundleName)
		if name == "" {
			continue
		}

		url := fmt.Sprintf(baseImageURL, name, name)
		filename := name + ".png"
		outFile := filepath.Join(outDir, filename)

		if err := downloadFileWithRetry(client, url, outFile, 3); err != nil {
			fmt.Printf("skip id=%d name=%s err=%v\n", row.ID, name, err)
			continue
		}

		mapping = append(mapping, coverMapRow{
			ID:              row.ID,
			AssetBundleName: name,
			URL:             url,
			File:            filename,
		})
		okCount++
		fmt.Printf("saved %s\n", filename)
		time.Sleep(80 * time.Millisecond)
	}

	mapBytes, _ := json.MarshalIndent(mapping, "", "  ")
	_ = os.WriteFile(filepath.Join(outDir, "_cover_map.json"), mapBytes, 0o644)

	fmt.Printf("done, saved=%d/%d\n", okCount, len(rows))
}

func loadMusicRows(file string) ([]musicRow, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var rows []musicRow
	if err := json.Unmarshal(b, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func downloadFileWithRetry(client *http.Client, url, outFile string, maxRetry int) error {
	var lastErr error
	for i := 0; i < maxRetry; i++ {
		if err := downloadFile(client, url, outFile); err == nil {
			return nil
		} else {
			lastErr = err
			time.Sleep(time.Duration(i+1) * 250 * time.Millisecond)
		}
	}
	return lastErr
}

func downloadFile(client *http.Client, url, outFile string) error {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 pjsk-cover-fetcher-go")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http %d", resp.StatusCode)
	}

	tmpFile := outFile + ".tmp"
	f, err := os.Create(tmpFile)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
		_ = os.Remove(tmpFile)
	}()

	if _, err = io.Copy(f, resp.Body); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}

	// Validate png integrity before replacing target file.
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if _, err = png.DecodeConfig(f); err != nil {
		return fmt.Errorf("invalid png: %w", err)
	}

	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmpFile, outFile); err != nil {
		return err
	}
	return nil
}

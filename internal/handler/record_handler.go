package handler

import (
	"fmt"
	"image"
	"image/color"
	stddraw "image/draw"
	_ "image/jpeg"
	"image/png"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"Project_sekai_search/internal/dto"
	"Project_sekai_search/internal/pkg/response"
	"Project_sekai_search/internal/service"

	"github.com/gin-gonic/gin"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	_ "golang.org/x/image/webp"
)

func UploadRecordHandler(recordSvc *service.RecordService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.UploadRecordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, http.StatusBadRequest, "req format error")
			return
		}

		record, created, avgB30, err := recordSvc.UploadRecord(c.Request.Context(), c.GetUint("user_id"), req)
		if err != nil {
			writeErr(c, err)
			return
		}

		response.OK(c, gin.H{
			"created": created,
			"record":  record,
			"avg_b30": avgB30,
		})
	}
}

func DeleteRecordHandler(recordSvc *service.RecordService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.DeleteRecordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, http.StatusBadRequest, "req format error")
			return
		}

		deleted, avgB30, err := recordSvc.DeleteRecord(c.Request.Context(), c.GetUint("user_id"), req)
		if err != nil {
			writeErr(c, err)
			return
		}

		response.OK(c, gin.H{
			"deleted": deleted,
			"avg_b30": avgB30,
		})
	}
}

func GetBest30Handler(recordSvc *service.RecordService) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, avgB30, err := recordSvc.GetBest30(c.Request.Context(), c.GetUint("user_id"), c.Query("calc_mode"))
		if err != nil {
			writeErr(c, err)
			return
		}

		response.OK(c, gin.H{
			"list":    items,
			"avg_b30": avgB30,
		})
	}
}

func GetB30TrendHandler(recordSvc *service.RecordService) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := recordSvc.GetB30Trend(c.Request.Context(), c.GetUint("user_id"), c.Query("calc_mode"))
		if err != nil {
			writeErr(c, err)
			return
		}

		response.OK(c, gin.H{
			"list": items,
		})
	}
}

func ExportB30ImageHandler(recordSvc *service.RecordService, userSvc *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.GetUint("user_id")
		calcMode := strings.TrimSpace(c.Query("calc_mode"))
		items, avgB30, err := recordSvc.GetBest30(c.Request.Context(), uid, calcMode)
		if err != nil {
			writeErr(c, err)
			return
		}
		profile, err := userSvc.GetMyProfile(c.Request.Context(), uid)
		if err != nil {
			writeErr(c, err)
			return
		}

		img := renderB30Image(profile, avgB30, items, strings.EqualFold(calcMode, "const"))
		c.Header("Content-Type", "image/png")
		c.Header("Content-Disposition", "inline; filename=b30.png")
		c.Header("Cache-Control", "no-store, max-age=0")
		c.Header("Pragma", "no-cache")
		_ = png.Encode(c.Writer, img)
	}
}

func GetRecordStatusesHandler(recordSvc *service.RecordService) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := recordSvc.GetUserRecordStatuses(c.Request.Context(), c.GetUint("user_id"))
		if err != nil {
			writeErr(c, err)
			return
		}

		response.OK(c, gin.H{
			"list": items,
		})
	}
}

func GetAchievementMapHandler(recordSvc *service.RecordService) gin.HandlerFunc {
	return func(c *gin.Context) {
		m, err := recordSvc.GetAchievementMap(c.Request.Context())
		if err != nil {
			writeErr(c, err)
			return
		}

		response.OK(c, gin.H{
			"map": m,
		})
	}
}

func GetRecordStatisticsHandler(recordSvc *service.RecordService) gin.HandlerFunc {
	return func(c *gin.Context) {
		difficulty := strings.TrimSpace(c.Query("difficulty"))
		mode := strings.TrimSpace(c.Query("mode"))
		minLevel, err := parseOptionalPositiveUint(c.Query("min_level"))
		if err != nil {
			response.Error(c, http.StatusBadRequest, "min_level format error")
			return
		}
		maxLevel, err := parseOptionalPositiveUint(c.Query("max_level"))
		if err != nil {
			response.Error(c, http.StatusBadRequest, "max_level format error")
			return
		}

		data, err := recordSvc.GetStatistics(c.Request.Context(), c.GetUint("user_id"), difficulty, mode, minLevel, maxLevel)
		if err != nil {
			writeErr(c, err)
			return
		}

		response.OK(c, gin.H{
			"difficulty":   data.Difficulty,
			"mode":         data.Mode,
			"min_level":    data.MinLevel,
			"max_level":    data.MaxLevel,
			"total_charts": data.TotalCharts,
			"buckets":      data.Buckets,
		})
	}
}

func parseOptionalPositiveUint(raw string) (uint, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	if n <= 0 {
		return 0, fmt.Errorf("value must be positive")
	}
	return uint(n), nil
}

func renderB30Image(profile *dto.MyProfileResponse, avg float64, list []dto.Best30Item, useConstDisplay bool) *image.RGBA {
	const (
		width       = 1920
		height      = 1080
		leftPanelW  = 520
		outerMargin = 28
		gridGap     = 12
		gridCols    = 5
	)

	if len(list) > 30 {
		list = list[:30]
	}

	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	drawVerticalGradient(canvas, canvas.Bounds(), color.RGBA{8, 10, 17, 255}, color.RGBA{17, 21, 33, 255})
	drawDiagonalNoise(canvas, canvas.Bounds(), color.RGBA{255, 255, 255, 14}, 8)

	fonts := getB30Fonts()

	leftRect := image.Rect(outerMargin, outerMargin, outerMargin+leftPanelW, height-outerMargin)
	drawRect(canvas, leftRect, color.RGBA{12, 16, 26, 240}, stddraw.Over)
	drawRect(canvas, image.Rect(leftRect.Min.X, leftRect.Min.Y, leftRect.Max.X, leftRect.Min.Y+8), color.RGBA{74, 139, 255, 255}, stddraw.Over)

	drawText(canvas, fonts.brandFace, leftRect.Min.X+26, leftRect.Min.Y+54, "Project Sekai B30", color.RGBA{236, 244, 255, 255})
	drawText(canvas, fonts.smallFace, leftRect.Min.X+26, leftRect.Min.Y+80, "Generated by local server", color.RGBA{139, 167, 210, 255})

	avatarRect := image.Rect(leftRect.Min.X+26, leftRect.Min.Y+114, leftRect.Min.X+206, leftRect.Min.Y+294)
	drawRect(canvas, avatarRect, color.RGBA{32, 42, 64, 255}, stddraw.Over)
	if avatar := loadAvatar(profile.AvatarURL); avatar != nil {
		drawImageCover(canvas, avatar, avatarRect)
	}
	drawRect(canvas, image.Rect(avatarRect.Min.X, avatarRect.Max.Y-6, avatarRect.Max.X, avatarRect.Max.Y), color.RGBA{88, 157, 255, 255}, stddraw.Over)

	name := clampTextByWidth(fonts.nameFace, safeText(profile.Username, "Player"), leftRect.Dx()-260)
	drawText(canvas, fonts.nameFace, leftRect.Min.X+232, leftRect.Min.Y+176, name, color.RGBA{250, 252, 255, 255})
	drawText(canvas, fonts.smallFace, leftRect.Min.X+232, leftRect.Min.Y+208, fmt.Sprintf("User ID: %d", profile.ID), color.RGBA{154, 177, 214, 255})
	drawText(canvas, fonts.smallFace, leftRect.Min.X+232, leftRect.Min.Y+236, fmt.Sprintf("Records: %d", len(list)), color.RGBA{154, 177, 214, 255})

	statCard := image.Rect(leftRect.Min.X+26, leftRect.Min.Y+330, leftRect.Max.X-26, leftRect.Min.Y+520)
	drawRect(canvas, statCard, color.RGBA{18, 28, 45, 230}, stddraw.Over)
	drawText(canvas, fonts.labelFace, statCard.Min.X+22, statCard.Min.Y+44, "RATING", color.RGBA{138, 177, 255, 255})
	drawTextOutline(canvas, fonts.bigFace, statCard.Min.X+22, statCard.Min.Y+116, fmt.Sprintf("%.4f", avg), color.RGBA{245, 250, 255, 255}, color.RGBA{0, 0, 0, 210})
	best15 := computeAverageByN(list, 15)
	drawTextShadow(canvas, fonts.avg15Face, statCard.Min.X+22, statCard.Min.Y+176, fmt.Sprintf("Best 15 Avg. %.4f", best15), color.RGBA{143, 176, 232, 255}, color.RGBA{0, 0, 0, 180})

	charRect := image.Rect(leftRect.Min.X+26, statCard.Max.Y+24, leftRect.Max.X-26, leftRect.Max.Y-26)
	drawRoundedRect(canvas, charRect, 18, color.RGBA{15, 26, 42, 240})
	if characterImg := loadCharacterImage(profile.CharacterImageURL); characterImg != nil {
		drawImageCoverTop(canvas, characterImg, charRect)
	}

	gridRect := image.Rect(leftRect.Max.X+22, outerMargin, width-outerMargin, height-outerMargin)
	drawRect(canvas, gridRect, color.RGBA{10, 14, 24, 210}, stddraw.Over)
	drawRect(canvas, image.Rect(gridRect.Min.X, gridRect.Min.Y, gridRect.Max.X, gridRect.Min.Y+8), color.RGBA{88, 157, 255, 255}, stddraw.Over)

	drawText(canvas, fonts.titleFace, gridRect.Min.X+20, gridRect.Min.Y+56, "BEST 30", color.RGBA{242, 248, 255, 255})
	drawTextShadow(canvas, fonts.smallFace, gridRect.Min.X+250, gridRect.Min.Y+54, fmt.Sprintf("AVG %.4f", avg), color.RGBA{173, 197, 236, 255}, color.RGBA{0, 0, 0, 180})

	gridTop := gridRect.Min.Y + 76
	gridContentH := gridRect.Max.Y - gridTop - 20
	gridRows := 1
	if len(list) > 0 {
		gridRows = (len(list) + gridCols - 1) / gridCols
	}
	cellW := (gridRect.Dx() - (gridGap * (gridCols + 1))) / gridCols
	cellH := (gridContentH - (gridGap * (gridRows + 1))) / gridRows
	if cellH < 110 {
		cellH = 110
	}

	for i, it := range list {
		col := i % gridCols
		row := i / gridCols
		x0 := gridRect.Min.X + gridGap + col*(cellW+gridGap)
		y0 := gridTop + gridGap + row*(cellH+gridGap)
		card := image.Rect(x0, y0, x0+cellW, y0+cellH)
		drawRect(canvas, card, color.RGBA{23, 31, 48, 255}, stddraw.Over)
		drawRect(canvas, image.Rect(card.Min.X, card.Max.Y-4, card.Max.X, card.Max.Y), difficultyColor(it.MusicDifficulty), stddraw.Over)

		if cover := loadCoverByAsset(it.AssetBundleName); cover != nil {
			drawImageCover(canvas, cover, card)
		}

		drawVerticalGradientOver(canvas, image.Rect(card.Min.X, card.Min.Y, card.Max.X, card.Min.Y+102), color.RGBA{0, 0, 0, 168}, color.RGBA{0, 0, 0, 18})
		drawRect(canvas, image.Rect(card.Min.X, card.Max.Y-46, card.Max.X, card.Max.Y), color.RGBA{0, 0, 0, 118}, stddraw.Over)

		title := clampTextByWidth(fonts.cardTitleFace, safeText(it.Title, "-"), cellW-18)
		drawTextShadow(canvas, fonts.cardTitleFace, card.Min.X+9, card.Min.Y+27, title, color.RGBA{245, 250, 255, 255}, color.RGBA{0, 0, 0, 180})
		drawTextShadow(canvas, fonts.rankFace, card.Max.X-58, card.Min.Y+27, fmt.Sprintf("#%02d", it.Rank), color.RGBA{232, 238, 255, 255}, color.RGBA{0, 0, 0, 180})

		scoreText := fmt.Sprintf("%.2f", it.ScoreValue)
		drawTextOutline(canvas, fonts.cardScoreFace, card.Min.X+12, card.Min.Y+80, scoreText, color.RGBA{255, 255, 255, 255}, color.RGBA{0, 0, 0, 226})

		diffText := difficultyLabel(it.MusicDifficulty)
		diffTag := fmt.Sprintf("%s %s", diffText, formatLevelDisplay(it.PlayLevel, it.ConstValue, useConstDisplay))
		status := normalizeAchievement(it.MusicAchievement)
		statusLabel := achievementLabel(status)

		badgeX := card.Min.X + 7
		badgeY := card.Max.Y - 38
		badgeH := 28
		badgeGap := 4
		badgeTotalW := cellW - 14
		badgeFace, diffW, statusW := calculateBadgeWidths(fonts, diffTag, statusLabel, badgeTotalW, badgeGap)

		diffRect := image.Rect(badgeX, badgeY, badgeX+diffW, badgeY+badgeH)
		drawRoundedRect(canvas, diffRect, 10, difficultyColor(it.MusicDifficulty))
		drawCenteredTextInRect(canvas, badgeFace, diffRect, clampTextByWidth(badgeFace, diffTag, diffRect.Dx()-10), difficultyTextColor(it.MusicDifficulty))

		statusRect := image.Rect(diffRect.Max.X+badgeGap, badgeY, diffRect.Max.X+badgeGap+statusW, badgeY+badgeH)
		drawAchievementBadge(canvas, statusRect, status, 10)
		drawCenteredTextInRect(canvas, badgeFace, statusRect, clampTextByWidth(badgeFace, statusLabel, statusRect.Dx()-10), achievementTextColor(status))
	}

	return canvas
}

func drawRect(img *image.RGBA, rect image.Rectangle, c color.Color, op stddraw.Op) {
	stddraw.Draw(img, rect, &image.Uniform{c}, image.Point{}, op)
}

func drawText(img *image.RGBA, face font.Face, x, y int, s string, c color.Color) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(s)
}

func drawTextShadow(img *image.RGBA, face font.Face, x, y int, s string, fg color.Color, shadow color.Color) {
	drawText(img, face, x+2, y+2, s, shadow)
	drawText(img, face, x, y, s, fg)
}

func drawTextOutline(img *image.RGBA, face font.Face, x, y int, s string, fg color.Color, outline color.Color) {
	offsets := [][2]int{{-2, 0}, {2, 0}, {0, -2}, {0, 2}, {-1, -1}, {1, -1}, {-1, 1}, {1, 1}}
	for _, of := range offsets {
		drawText(img, face, x+of[0], y+of[1], s, outline)
	}
	drawText(img, face, x, y, s, fg)
}

func drawTextGlow(img *image.RGBA, face font.Face, x, y int, s string, fg color.Color, outline color.Color, glow color.Color) {
	glowOffsets := [][2]int{
		{-3, 0}, {3, 0}, {0, -3}, {0, 3},
		{-2, -2}, {2, -2}, {-2, 2}, {2, 2},
	}
	for _, of := range glowOffsets {
		drawText(img, face, x+of[0], y+of[1], s, glow)
	}
	drawTextOutline(img, face, x, y, s, fg, outline)
}

func drawCenteredTextInRect(img *image.RGBA, face font.Face, rect image.Rectangle, s string, c color.Color) {
	if rect.Dx() <= 0 || rect.Dy() <= 0 || strings.TrimSpace(s) == "" {
		return
	}
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: face,
	}
	w := d.MeasureString(s).Ceil()
	metrics := face.Metrics()
	ascent := metrics.Ascent.Ceil()
	descent := metrics.Descent.Ceil()
	x := rect.Min.X + (rect.Dx()-w)/2
	y := rect.Min.Y + (rect.Dy()+ascent-descent)/2
	d.Dot = fixed.P(x, y)
	d.DrawString(s)
}

func measureTextWidth(face font.Face, s string) int {
	d := font.Drawer{Face: face}
	return d.MeasureString(s).Ceil()
}

func safeText(s, fallback string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	return s
}

func loadAvatar(avatarURL string) image.Image {
	avatarURL = strings.TrimSpace(avatarURL)
	if avatarURL == "" {
		return nil
	}
	local := strings.TrimPrefix(avatarURL, "/")
	if !strings.HasPrefix(local, "static/") {
		return nil
	}
	f, err := os.Open(filepath.Clean(local))
	if err != nil {
		return nil
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil
	}
	return img
}

func loadCharacterImage(imageURL string) image.Image {
	imageURL = strings.TrimSpace(imageURL)
	if imageURL == "" {
		return nil
	}

	if strings.HasPrefix(strings.ToLower(imageURL), "http://") || strings.HasPrefix(strings.ToLower(imageURL), "https://") {
		client := &http.Client{Timeout: 4 * time.Second}
		resp, err := client.Get(imageURL)
		if err != nil {
			return nil
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil
		}
		img, _, err := image.Decode(resp.Body)
		if err != nil {
			return nil
		}
		return img
	}

	local := strings.TrimPrefix(imageURL, "/")
	local, err := url.PathUnescape(local)
	if err != nil {
		local = strings.TrimPrefix(imageURL, "/")
	}
	local = filepath.Clean(filepath.FromSlash(local))
	if !strings.HasPrefix(filepath.ToSlash(local), "static/") {
		return nil
	}

	f, err := os.Open(local)
	if err != nil {
		return nil
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil
	}
	return img
}

func loadCoverByAsset(asset string) image.Image {
	asset = strings.TrimSpace(asset)
	if asset == "" {
		return nil
	}
	file := filepath.Join("static", "assets", filepath.Base(asset)+".png")
	f, err := os.Open(file)
	if err != nil {
		return nil
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil
	}
	return img
}

func drawImageCover(dst *image.RGBA, src image.Image, rect image.Rectangle) {
	b := src.Bounds()
	if b.Dx() == 0 || b.Dy() == 0 || rect.Dx() <= 0 || rect.Dy() <= 0 {
		return
	}
	srcRatio := float64(b.Dx()) / float64(b.Dy())
	dstRatio := float64(rect.Dx()) / float64(rect.Dy())
	var crop image.Rectangle
	if srcRatio > dstRatio {
		targetW := int(float64(b.Dy()) * dstRatio)
		if targetW < 1 {
			targetW = 1
		}
		x0 := b.Min.X + (b.Dx()-targetW)/2
		crop = image.Rect(x0, b.Min.Y, x0+targetW, b.Max.Y)
	} else {
		targetH := int(float64(b.Dx()) / dstRatio)
		if targetH < 1 {
			targetH = 1
		}
		y0 := b.Min.Y + (b.Dy()-targetH)/2
		crop = image.Rect(b.Min.X, y0, b.Max.X, y0+targetH)
	}
	xdraw.CatmullRom.Scale(dst, rect, src, crop, stddraw.Over, nil)
}

func drawImageCoverTop(dst *image.RGBA, src image.Image, rect image.Rectangle) {
	b := src.Bounds()
	if b.Dx() == 0 || b.Dy() == 0 || rect.Dx() <= 0 || rect.Dy() <= 0 {
		return
	}
	srcRatio := float64(b.Dx()) / float64(b.Dy())
	dstRatio := float64(rect.Dx()) / float64(rect.Dy())
	var crop image.Rectangle
	if srcRatio > dstRatio {
		targetW := int(float64(b.Dy()) * dstRatio)
		if targetW < 1 {
			targetW = 1
		}
		x0 := b.Min.X + (b.Dx()-targetW)/2
		crop = image.Rect(x0, b.Min.Y, x0+targetW, b.Max.Y)
	} else {
		targetH := int(float64(b.Dx()) / dstRatio)
		if targetH < 1 {
			targetH = 1
		}
		// Top-aligned crop for portraits, so character head is kept visible.
		crop = image.Rect(b.Min.X, b.Min.Y, b.Max.X, b.Min.Y+targetH)
	}
	xdraw.CatmullRom.Scale(dst, rect, src, crop, stddraw.Over, nil)
}

func drawVerticalGradient(img *image.RGBA, rect image.Rectangle, top color.RGBA, bottom color.RGBA) {
	drawVerticalGradientWithOp(img, rect, top, bottom, stddraw.Src)
}

func drawVerticalGradientOver(img *image.RGBA, rect image.Rectangle, top color.RGBA, bottom color.RGBA) {
	drawVerticalGradientWithOp(img, rect, top, bottom, stddraw.Over)
}

func drawVerticalGradientWithOp(img *image.RGBA, rect image.Rectangle, top color.RGBA, bottom color.RGBA, op stddraw.Op) {
	h := rect.Dy()
	if h < 1 {
		h = 1
	}
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		t := 0.0
		if h > 1 {
			t = float64(y-rect.Min.Y) / float64(h-1)
		}
		c := color.RGBA{
			R: uint8(float64(top.R)*(1-t) + float64(bottom.R)*t),
			G: uint8(float64(top.G)*(1-t) + float64(bottom.G)*t),
			B: uint8(float64(top.B)*(1-t) + float64(bottom.B)*t),
			A: uint8(float64(top.A)*(1-t) + float64(bottom.A)*t),
		}
		drawRect(img, image.Rect(rect.Min.X, y, rect.Max.X, y+1), c, op)
	}
}

func drawDiagonalNoise(img *image.RGBA, rect image.Rectangle, c color.RGBA, step int) {
	if step < 2 {
		step = 2
	}
	for x := rect.Min.X - rect.Dy(); x < rect.Max.X; x += step {
		for y := rect.Min.Y; y < rect.Max.Y; y++ {
			px := x + (y - rect.Min.Y)
			if px >= rect.Min.X && px < rect.Max.X {
				img.Set(px, y, c)
			}
		}
	}
}

func clampTextByWidth(face font.Face, s string, maxWidth int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	if maxWidth <= 12 {
		return "..."
	}
	d := font.Drawer{Face: face}
	if d.MeasureString(s).Ceil() <= maxWidth {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		try := string(runes) + "..."
		if d.MeasureString(try).Ceil() <= maxWidth {
			return try
		}
	}
	return "..."
}

func formatLevelDisplay(playLevel uint, constValue float64, useConstDisplay bool) string {
	if useConstDisplay && constValue > 0 {
		return fmt.Sprintf("%.1f", constValue)
	}
	return fmt.Sprintf("%d", playLevel)
}

func computeAverageByN(list []dto.Best30Item, n int) float64 {
	if n <= 0 || len(list) == 0 {
		return 0
	}
	if len(list) < n {
		n = len(list)
	}
	total := 0.0
	for i := 0; i < n; i++ {
		total += list[i].ScoreValue
	}
	return total / float64(n)
}

func difficultyColor(diff string) color.RGBA {
	switch strings.ToLower(strings.TrimSpace(diff)) {
	case "easy":
		return color.RGBA{25, 220, 142, 255}
	case "normal":
		return color.RGBA{65, 207, 255, 255}
	case "hard":
		return color.RGBA{255, 215, 51, 255}
	case "expert":
		return color.RGBA{255, 86, 151, 255}
	case "matser", "master":
		return color.RGBA{173, 118, 255, 255}
	case "append":
		return color.RGBA{214, 149, 255, 255}
	default:
		return color.RGBA{160, 173, 193, 255}
	}
}

func difficultyLabel(diff string) string {
	switch strings.ToLower(strings.TrimSpace(diff)) {
	case "easy":
		return "EASY"
	case "normal":
		return "NORMAL"
	case "hard":
		return "HARD"
	case "expert":
		return "EXPERT"
	case "matser", "master":
		return "MASTER"
	case "append":
		return "APPEND"
	default:
		return strings.ToUpper(safeText(diff, "UNKNOWN"))
	}
}

func difficultyTextColor(diff string) color.RGBA {
	switch strings.ToLower(strings.TrimSpace(diff)) {
	case "hard", "append":
		return color.RGBA{24, 24, 24, 255}
	default:
		return color.RGBA{255, 255, 255, 255}
	}
}

func normalizeAchievement(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case strings.Contains(s, "all_perfect"), strings.Contains(s, "all perfect"), s == "ap":
		return "all_perfect"
	case strings.Contains(s, "full_combo"), strings.Contains(s, "full combo"), s == "fc":
		return "full_combo"
	case strings.Contains(s, "clear"):
		return "clear"
	default:
		return "not_played"
	}
}

func achievementLabel(key string) string {
	switch key {
	case "all_perfect":
		return "ALL PERFECT"
	case "full_combo":
		return "FULL COMBO"
	case "clear":
		return "CLEAR"
	default:
		return "NOT PLAYED"
	}
}

func achievementTextColor(key string) color.RGBA {
	switch key {
	case "clear":
		return color.RGBA{26, 26, 26, 255}
	case "full_combo":
		return color.RGBA{50, 27, 85, 255}
	case "all_perfect":
		return color.RGBA{24, 49, 76, 255}
	default:
		return color.RGBA{45, 66, 102, 255}
	}
}

func drawAchievementBadge(canvas *image.RGBA, rect image.Rectangle, key string, radius int) {
	switch key {
	case "clear":
		drawRoundedRect(canvas, rect, radius, difficultyColor("hard"))
	case "full_combo":
		drawRoundedRect(canvas, rect, radius, color.RGBA{211, 173, 255, 255})
	case "all_perfect":
		drawRoundedHorizontalGradient(canvas, rect, radius,
			color.RGBA{255, 248, 215, 255},
			color.RGBA{255, 227, 247, 255},
			color.RGBA{191, 251, 255, 255},
		)
	default:
		drawRoundedRect(canvas, rect, radius, color.RGBA{199, 211, 235, 255})
	}
}

func drawHorizontalGradient(img *image.RGBA, rect image.Rectangle, left, mid, right color.RGBA) {
	w := max(1, rect.Dx())
	for x := rect.Min.X; x < rect.Max.X; x++ {
		t := float64(x-rect.Min.X) / float64(w-1)
		var c color.RGBA
		if t <= 0.5 {
			k := t * 2
			c = color.RGBA{
				R: uint8(float64(left.R)*(1-k) + float64(mid.R)*k),
				G: uint8(float64(left.G)*(1-k) + float64(mid.G)*k),
				B: uint8(float64(left.B)*(1-k) + float64(mid.B)*k),
				A: 255,
			}
		} else {
			k := (t - 0.5) * 2
			c = color.RGBA{
				R: uint8(float64(mid.R)*(1-k) + float64(right.R)*k),
				G: uint8(float64(mid.G)*(1-k) + float64(right.G)*k),
				B: uint8(float64(mid.B)*(1-k) + float64(right.B)*k),
				A: 255,
			}
		}
		drawRect(img, image.Rect(x, rect.Min.Y, x+1, rect.Max.Y), c, stddraw.Src)
	}
}

func drawRoundedHorizontalGradient(img *image.RGBA, rect image.Rectangle, radius int, left, mid, right color.RGBA) {
	mask := roundedRectMask(rect, radius)
	gradient := image.NewRGBA(rect)
	drawHorizontalGradient(gradient, rect, left, mid, right)
	stddraw.DrawMask(img, rect, gradient, rect.Min, mask, rect.Min, stddraw.Over)
}

func drawRoundedRectWithBorder(img *image.RGBA, rect image.Rectangle, radius int, fill color.RGBA, border color.RGBA) {
	if rect.Dx() <= 0 || rect.Dy() <= 0 {
		return
	}
	drawRoundedRect(img, rect, radius, border)
	inner := image.Rect(rect.Min.X+1, rect.Min.Y+1, rect.Max.X-1, rect.Max.Y-1)
	if inner.Dx() > 0 && inner.Dy() > 0 {
		drawRoundedRect(img, inner, max(0, radius-1), fill)
	}
}

func drawRoundedRect(img *image.RGBA, rect image.Rectangle, radius int, c color.RGBA) {
	if rect.Dx() <= 0 || rect.Dy() <= 0 {
		return
	}
	mask := roundedRectMask(rect, radius)
	stddraw.DrawMask(img, rect, &image.Uniform{C: c}, image.Point{}, mask, rect.Min, stddraw.Over)
}

func roundedRectMask(rect image.Rectangle, radius int) *image.Alpha {
	mask := image.NewAlpha(rect)
	w := rect.Dx()
	h := rect.Dy()
	r := min(radius, min(w/2, h/2))
	if r <= 0 {
		for y := rect.Min.Y; y < rect.Max.Y; y++ {
			for x := rect.Min.X; x < rect.Max.X; x++ {
				mask.SetAlpha(x, y, color.Alpha{A: 255})
			}
		}
		return mask
	}
	rr := r * r
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			rx := x - rect.Min.X
			ry := y - rect.Min.Y
			inside := false
			switch {
			case rx >= r && rx < w-r:
				inside = true
			case ry >= r && ry < h-r:
				inside = true
			default:
				var cx, cy int
				if rx < r {
					cx = r - 1
				} else {
					cx = w - r
				}
				if ry < r {
					cy = r - 1
				} else {
					cy = h - r
				}
				dx := rx - cx
				dy := ry - cy
				inside = dx*dx+dy*dy <= rr
			}
			if inside {
				mask.SetAlpha(x, y, color.Alpha{A: 255})
			}
		}
	}
	return mask
}

type b30FontPack struct {
	titleFace     font.Face
	brandFace     font.Face
	nameFace      font.Face
	bigFace       font.Face
	labelFace     font.Face
	metaFace      font.Face
	avg15Face     font.Face
	cardTitleFace font.Face
	cardScoreFace font.Face
	cardMetaFace  font.Face
	cardMetaSmall font.Face
	rankFace      font.Face
	smallFace     font.Face
}

func getB30Fonts() *b30FontPack {
	return loadB30FontPack()
}

func loadB30FontPack() *b30FontPack {
	regularBytes := loadFirstExistingFontBytes(regularFontPaths())
	boldBytes := loadFirstExistingFontBytes(boldFontPaths())
	latinBytes := loadFirstExistingFontBytes(latinFontPaths())

	if len(regularBytes) == 0 {
		regularBytes = goregular.TTF
	}
	if len(boldBytes) == 0 {
		boldBytes = gobold.TTF
	}

	regularFaceFactory := makeFaceFactory(regularBytes)
	boldFaceFactory := makeFaceFactory(boldBytes)
	latinFaceFactory := makeFaceFactory(latinBytes)
	if latinFaceFactory == nil {
		latinFaceFactory = boldFaceFactory
	}

	if regularFaceFactory == nil || boldFaceFactory == nil {
		return &b30FontPack{
			titleFace:     basicfont.Face7x13,
			brandFace:     basicfont.Face7x13,
			nameFace:      basicfont.Face7x13,
			bigFace:       basicfont.Face7x13,
			labelFace:     basicfont.Face7x13,
			metaFace:      basicfont.Face7x13,
			avg15Face:     basicfont.Face7x13,
			cardTitleFace: basicfont.Face7x13,
			cardScoreFace: basicfont.Face7x13,
			cardMetaFace:  basicfont.Face7x13,
			cardMetaSmall: basicfont.Face7x13,
			rankFace:      basicfont.Face7x13,
			smallFace:     basicfont.Face7x13,
		}
	}

	return &b30FontPack{
		titleFace:     latinFaceFactory(44),
		brandFace:     latinFaceFactory(42),
		nameFace:      boldFaceFactory(44),
		bigFace:       latinFaceFactory(60),
		labelFace:     boldFaceFactory(24),
		metaFace:      regularFaceFactory(22),
		avg15Face:     latinFaceFactory(28),
		cardTitleFace: boldFaceFactory(17),
		cardScoreFace: latinFaceFactory(40),
		cardMetaFace:  regularFaceFactory(15),
		cardMetaSmall: regularFaceFactory(13),
		rankFace:      latinFaceFactory(24),
		smallFace:     regularFaceFactory(18),
	}
}

func calculateBadgeWidths(fonts *b30FontPack, diffTag, statusLabel string, totalW, gap int) (font.Face, int, int) {
	avail := max(0, totalW-gap)
	face := fonts.cardMetaFace
	diffNeed := measureTextWidth(face, diffTag) + 16
	statusNeed := measureTextWidth(face, statusLabel) + 16
	if diffNeed+statusNeed > avail {
		face = fonts.cardMetaSmall
		diffNeed = measureTextWidth(face, diffTag) + 14
		statusNeed = measureTextWidth(face, statusLabel) + 14
	}

	if diffNeed+statusNeed <= avail {
		rest := avail - (diffNeed + statusNeed)
		diffW := diffNeed + rest/2
		statusW := statusNeed + rest - rest/2
		return face, diffW, statusW
	}

	if avail <= 0 {
		return face, 0, 0
	}

	sum := max(1, diffNeed+statusNeed)
	diffW := int(float64(avail) * float64(diffNeed) / float64(sum))
	statusW := avail - diffW

	minDiff := 68
	minStatus := 80
	if diffW < minDiff {
		diffW = minDiff
		statusW = avail - diffW
	}
	if statusW < minStatus {
		statusW = minStatus
		diffW = avail - statusW
	}
	if diffW < 54 {
		diffW = 54
		statusW = avail - diffW
	}
	if statusW < 60 {
		statusW = 60
		diffW = avail - statusW
	}
	return face, max(0, diffW), max(0, statusW)
}

func makeFaceFactory(fontBytes []byte) func(float64) font.Face {
	ft, err := opentype.Parse(fontBytes)
	if err != nil {
		col, colErr := opentype.ParseCollection(fontBytes)
		if colErr != nil || col.NumFonts() == 0 {
			return nil
		}
		// Many Windows fonts are TTC collections; use the first usable face.
		f0, fErr := col.Font(0)
		if fErr != nil {
			return nil
		}
		ft = f0
	}
	return func(size float64) font.Face {
		face, e := opentype.NewFace(ft, &opentype.FaceOptions{
			Size:    size,
			DPI:     72,
			Hinting: font.HintingFull,
		})
		if e != nil {
			return basicfont.Face7x13
		}
		return face
	}
}

func loadFirstExistingFontBytes(candidates []string) []byte {
	for _, p := range candidates {
		b, err := os.ReadFile(filepath.Clean(p))
		if err == nil && len(b) > 0 {
			return b
		}
	}
	return nil
}

func regularFontPaths() []string {
	paths := []string{
		filepath.Join("static", "assets", "fonts", "NotoSansJP-Regular.ttf"),
		filepath.Join("static", "assets", "fonts", "NotoSansSC-Regular.ttf"),
		filepath.Join("static", "assets", "fonts", "NotoSansCJKjp-Regular.otf"),
		filepath.Join("static", "assets", "fonts", "SourceHanSansSC-Regular.otf"),
		filepath.Join("static", "assets", "fonts", "SourceHanSansJP-Regular.otf"),
		filepath.Join("static", "assets", "fonts", "SourceHanSansCN-Regular.otf"),
	}
	if runtime.GOOS == "windows" {
		paths = append(paths,
			`C:\Windows\Fonts\meiryo.ttc`,
			`C:\Windows\Fonts\YuGothM.ttc`,
			`C:\Windows\Fonts\YuGothR.ttc`,
			`C:\Windows\Fonts\msgothic.ttc`,
			`C:\Windows\Fonts\msmincho.ttc`,
			`C:\Windows\Fonts\msyh.ttf`,
			`C:\Windows\Fonts\simhei.ttf`,
		)
	} else {
		paths = append(paths,
			`/usr/share/fonts/opentype/noto/NotoSansCJKjp-Regular.otf`,
			`/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc`,
			`/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc`,
		)
	}
	return paths
}

func boldFontPaths() []string {
	paths := []string{
		filepath.Join("static", "assets", "fonts", "NotoSansJP-Bold.ttf"),
		filepath.Join("static", "assets", "fonts", "NotoSansSC-Bold.ttf"),
		filepath.Join("static", "assets", "fonts", "NotoSansCJKjp-Bold.otf"),
		filepath.Join("static", "assets", "fonts", "SourceHanSansSC-Bold.otf"),
		filepath.Join("static", "assets", "fonts", "SourceHanSansJP-Bold.otf"),
		filepath.Join("static", "assets", "fonts", "SourceHanSansCN-Bold.otf"),
	}
	if runtime.GOOS == "windows" {
		paths = append(paths,
			`C:\Windows\Fonts\meiryob.ttc`,
			`C:\Windows\Fonts\YuGothB.ttc`,
			`C:\Windows\Fonts\msgothic.ttc`,
			`C:\Windows\Fonts\msyhbd.ttf`,
		)
	} else {
		paths = append(paths,
			`/usr/share/fonts/opentype/noto/NotoSansCJKjp-Bold.otf`,
			`/usr/share/fonts/truetype/noto/NotoSansCJK-Bold.ttc`,
			`/usr/share/fonts/opentype/noto/NotoSansCJK-Bold.ttc`,
		)
	}
	return paths
}

func latinFontPaths() []string {
	paths := []string{
		filepath.Join("static", "assets", "fonts", "Orbitron-Bold.ttf"),
		filepath.Join("static", "assets", "fonts", "Rajdhani-Bold.ttf"),
	}
	if runtime.GOOS == "windows" {
		paths = append(paths,
			`C:\Windows\Fonts\bahnschrift.ttf`,
			`C:\Windows\Fonts\segoeuib.ttf`,
			`C:\Windows\Fonts\arialbd.ttf`,
		)
	} else {
		paths = append(paths,
			`/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf`,
			`/usr/share/fonts/truetype/liberation2/LiberationSans-Bold.ttf`,
		)
	}
	return paths
}

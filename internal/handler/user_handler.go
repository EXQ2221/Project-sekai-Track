package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"Project_sekai_search/internal/dto"
	"Project_sekai_search/internal/pkg/response"
	"Project_sekai_search/internal/service"

	"github.com/gin-gonic/gin"
)

func RegisterHandler(userSvc *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, http.StatusBadRequest, "request format error")
			return
		}

		user, err := userSvc.RegisterService(c.Request.Context(), req)
		if err != nil {
			writeErr(c, err)
			return
		}

		response.OK(c, gin.H{
			"user_id":  user.ID,
			"username": user.Username,
			"email":    user.Email,
		})
	}
}

func LoginHandler(authSvc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, http.StatusBadRequest, "req format error")
			return
		}

		pair, user, deviceID, err := authSvc.Login(c.Request.Context(), req, c.ClientIP(), c.GetHeader("User-Agent"))
		if err != nil {
			writeErr(c, err)
			return
		}

		response.OK(c, gin.H{
			"user_id":            user.ID,
			"username":           user.Username,
			"token":              pair.AccessToken,
			"refresh_token":      pair.RefreshToken,
			"session_id":         pair.SessionID,
			"device_id":          deviceID,
			"access_expires_at":  pair.AccessExpiresAt,
			"refresh_expires_at": pair.RefreshExpiresAt,
		})
	}
}

func RefreshHandler(authSvc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.RefreshRequest
		if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
			response.Error(c, http.StatusBadRequest, "invalid refresh token")
			return
		}

		pair, err := authSvc.Refresh(c.Request.Context(), req, c.ClientIP(), c.GetHeader("User-Agent"))
		if err != nil {
			writeErr(c, err)
			return
		}

		response.OK(c, gin.H{
			"access_token":       pair.AccessToken,
			"refresh_token":      pair.RefreshToken,
			"session_id":         pair.SessionID,
			"access_expires_at":  pair.AccessExpiresAt,
			"refresh_expires_at": pair.RefreshExpiresAt,
		})
	}
}

func LogoutHandler(authSvc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		accessToken := c.GetString("access_token")
		if accessToken == "" {
			response.Error(c, http.StatusUnauthorized, "missing access token")
			return
		}

		if err := authSvc.Logout(c.Request.Context(), accessToken); err != nil {
			writeErr(c, err)
			return
		}

		response.OK(c, gin.H{"ok": true})
	}
}

func LogoutAllHandler(authSvc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.LogoutAllRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, http.StatusBadRequest, "req format error")
			return
		}

		if err := authSvc.LogoutAll(c.Request.Context(), c.GetUint("user_id"), req.Password); err != nil {
			writeErr(c, err)
			return
		}

		response.OK(c, gin.H{"ok": true})
	}
}

func ListSessionsHandler(authSvc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessions, err := authSvc.ListSessions(c.Request.Context(), c.GetUint("user_id"), c.GetString("session_id"))
		if err != nil {
			writeErr(c, err)
			return
		}

		response.OK(c, gin.H{"sessions": sessions})
	}
}

func RevokeSessionHandler(authSvc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.RevokeSessionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, http.StatusBadRequest, "req format error")
			return
		}

		if err := authSvc.RevokeSession(c.Request.Context(), c.GetUint("user_id"), req.SessionID, req.Password); err != nil {
			writeErr(c, err)
			return
		}

		response.OK(c, gin.H{"ok": true})
	}
}

func ChangePassHandler(userSvc *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.ChangePassRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, http.StatusBadRequest, "req format error")
			return
		}

		if err := userSvc.ChangePassService(c.Request.Context(), req, c.GetUint("user_id")); err != nil {
			writeErr(c, err)
			return
		}

		response.OK(c, gin.H{
			"ok":             true,
			"need_relogin":   true,
			"sessions_reset": true,
		})
	}
}

func GetMyProfileHandler(userSvc *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		profile, err := userSvc.GetMyProfile(c.Request.Context(), c.GetUint("user_id"))
		if err != nil {
			writeErr(c, err)
			return
		}

		response.OK(c, profile)
	}
}

func ListCharactersHandler(userSvc *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := userSvc.ListCharacters()
		if err != nil {
			writeErr(c, err)
			return
		}
		response.OK(c, gin.H{"list": items})
	}
}

func UpdateProfileHandler(userSvc *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.UpdateProfileRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, http.StatusBadRequest, "req format error")
			return
		}
		if err := userSvc.UpdateProfile(c.Request.Context(), c.GetUint("user_id"), req.Profile); err != nil {
			writeErr(c, err)
			return
		}
		response.OK(c, gin.H{"profile": req.Profile})
	}
}

func UpdateCharacterHandler(userSvc *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.UpdateCharacterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, http.StatusBadRequest, "req format error")
			return
		}
		if err := userSvc.UpdateCharacter(c.Request.Context(), c.GetUint("user_id"), req.Character); err != nil {
			writeErr(c, err)
			return
		}
		response.OK(c, gin.H{"character": req.Character})
	}
}

func UploadAvatarHandler(userSvc *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("user_id")

		file, err := c.FormFile("avatar")
		if err != nil {
			response.Error(c, http.StatusBadRequest, "missing avatar file")
			return
		}

		const maxSize = 5 * 1024 * 1024
		if file.Size > maxSize {
			response.Error(c, http.StatusBadRequest, "file too large (max 5MB)")
			return
		}

		f, err := file.Open()
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "open file failed")
			return
		}
		defer f.Close()

		buf := make([]byte, 512)
		n, _ := f.Read(buf)
		contentType := http.DetectContentType(buf[:n])
		ext := ""
		switch contentType {
		case "image/jpeg":
			ext = ".jpg"
		case "image/png":
			ext = ".png"
		case "image/webp":
			ext = ".webp"
		default:
			response.Error(c, http.StatusBadRequest, "only jpg/png/webp allowed")
			return
		}

		saveDir := filepath.Join("static", "uploads", "avatar")
		if err := os.MkdirAll(saveDir, 0o755); err != nil {
			response.Error(c, http.StatusInternalServerError, "mkdir failed")
			return
		}

		filename := fmt.Sprintf("u%d_%d%s", userID, time.Now().UnixNano(), ext)
		savePath := filepath.Join(saveDir, filename)
		if err := c.SaveUploadedFile(file, savePath); err != nil {
			response.Error(c, http.StatusInternalServerError, "save file failed")
			return
		}

		avatarURL := "/static/uploads/avatar/" + filename
		if err := userSvc.UpdateAvatarURL(c.Request.Context(), userID, avatarURL); err != nil {
			writeErr(c, err)
			return
		}

		response.OK(c, gin.H{
			"avatar_url": avatarURL,
		})
	}
}

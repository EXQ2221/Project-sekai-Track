package handler

import (
	"net/http"
	"strconv"

	"Project_sekai_search/internal/dto"
	"Project_sekai_search/internal/pkg/response"
	"Project_sekai_search/internal/service"
	"github.com/gin-gonic/gin"
)

func ListMusicsHandler(musicSvc *service.MusicService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var q dto.ListMusicQuery
		if err := c.ShouldBindQuery(&q); err != nil {
			response.Error(c, http.StatusBadRequest, "query format error")
			return
		}

		items, total, page, size, err := musicSvc.ListMusics(c.Request.Context(), q)
		if err != nil {
			writeErr(c, err)
			return
		}

		response.OK(c, gin.H{
			"list":  items,
			"total": total,
			"page":  page,
			"size":  size,
		})
	}
}

func GetMusicDetailHandler(musicSvc *service.MusicService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id64, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil || id64 == 0 {
			response.Error(c, http.StatusBadRequest, "id format error")
			return
		}

		music, err := musicSvc.GetMusicDetail(c.Request.Context(), uint(id64))
		if err != nil {
			writeErr(c, err)
			return
		}

		response.OK(c, music)
	}
}

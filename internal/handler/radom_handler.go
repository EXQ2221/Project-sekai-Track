package handler

import (
	"Project_sekai_search/internal/dto"
	"Project_sekai_search/internal/pkg/response"
	"Project_sekai_search/internal/service"

	"github.com/gin-gonic/gin"
)

func RandomMusicRecommendation(randomSvc *service.RandomService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var q dto.RandomMusicRecommendationQuery
		userID := c.GetUint("user_id")
		q.CalcMode = c.Query("calc_mode")

		resp, err := randomSvc.RandomMusicRecommendation(c.Request.Context(), userID, q.CalcMode)
		if err != nil {
			writeErr(c, err)
			return
		}

		response.OK(c, resp)

	}
}

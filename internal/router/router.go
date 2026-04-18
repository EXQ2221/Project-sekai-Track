package router

import (
	"Project_sekai_search/internal/handler"
	"Project_sekai_search/internal/middleware"
	"Project_sekai_search/internal/service"

	"github.com/gin-gonic/gin"
)

func InitRouter(
	authService *service.AuthService,
	userService *service.UserService,
	musicService *service.MusicService,
	recordService *service.RecordService,
	randomService *service.RandomService,
) {
	r := gin.Default()
	r.Use(middleware.CORSMiddleware())
	r.Static("/static", "./static")
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "backend is running"})
	})

	public := r.Group("/")
	public.Use(middleware.RateLimit())
	{
		public.POST("/register", handler.RegisterHandler(userService))
		public.POST("/login", handler.LoginHandler(authService))

		public.POST("/refresh", handler.RefreshHandler(authService))

		public.GET("/characters", handler.ListCharactersHandler(userService))
		public.GET("/musics", handler.ListMusicsHandler(musicService))
		public.GET("/musics/:id", handler.GetMusicDetailHandler(musicService))
	}

	private := r.Group("/")
	private.Use(middleware.AuthMiddleware(authService))
	private.Use(middleware.RateLimit())
	{
		private.POST("/logout", handler.LogoutHandler(authService))
		private.POST("/logout-all", handler.LogoutAllHandler(authService))
		private.GET("/sessions", handler.ListSessionsHandler(authService))
		private.POST("/sessions/revoke", handler.RevokeSessionHandler(authService))
		private.POST("/change_pass", handler.ChangePassHandler(userService))

		private.GET("/me", handler.GetMyProfileHandler(userService))
		private.POST("/me/profile", handler.UpdateProfileHandler(userService))
		private.POST("/me/character", handler.UpdateCharacterHandler(userService))
		private.POST("/me/avatar", handler.UploadAvatarHandler(userService))

		private.POST("/records", handler.UploadRecordHandler(recordService))
		private.DELETE("/records", handler.DeleteRecordHandler(recordService))
		private.POST("/musics/:id/alias", handler.AddMusicAliasHandler(musicService))
		private.GET("/records/b30", handler.GetBest30Handler(recordService))
		private.GET("/records/b30/trend", handler.GetB30TrendHandler(recordService))
		private.GET("/records/b30/image", handler.ExportB30ImageHandler(recordService, userService))
		private.GET("/records/statuses", handler.GetRecordStatusesHandler(recordService))
		private.GET("/records/achievement-map", handler.GetAchievementMapHandler(recordService))
		private.GET("/records/statistics", handler.GetRecordStatisticsHandler(recordService))

		private.GET("/random/music", handler.RandomMusicRecommendation(randomService))
	}

	r.Run(":8080")
}

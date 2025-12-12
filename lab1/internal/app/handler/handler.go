package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "lab1/cmd/webserver/docs"
	"lab1/internal/app/config"
	"lab1/internal/app/repository"
)

type Handler struct {
	Repository *repository.Repository
	Config     *config.Config
}

func NewHandler(c *config.Config, r *repository.Repository) *Handler {
	return &Handler{
		Repository: r,
		Config:     c,
	}
}

func (h *Handler) RegisterHandler(router *gin.Engine) {
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/reaction", h.GetReactionsAPI)
	router.GET("/reaction/:id", h.GetReactionAPI)
	router.GET("/synthesis/:id", h.GetSynthesis)
	router.POST("/add-reaction-in-synthesis", h.AddReactionInSynthesis)
	router.POST("/delete/:id", h.RemoveSynthesis)

	//API
	authM := router.Group("")
	authM.Use(h.WithAuthCheck(true))
	authU := router.Group("")
	authU.Use(h.WithAuthCheck(false))
	authU5 := router.Group("")
	authU5.Use(h.WithAuthCheckLab5(false))
	//домен услуги (реакции)
	router.GET("/API/reaction", h.GetReactionsAPI)
	router.GET("/API/reaction/:id", h.GetReactionAPI)
	authM.POST("/API/create-reaction", h.CreateReactionAPI)
	authM.PUT("/API/reaction/:id", h.ChangeReactionAPI)
	authM.DELETE("/API/reaction/:id", h.DeleteReactionAPI)
	authU.POST("/API/reaction/:id/add-reaction-in-synthesis", h.AddReactionInSynthesisAPI)
	authM.POST("/API/reaction/:id/image", h.UploadReactionImageAPI)

	//домен заявки (синтез)
	authU.GET("/API/synthesis/icon", h.GetSynthesisIconAPI)
	//authU.GET("/API/synthesis/icon", h.GetSynthesisIconAPI)
	authU.GET("/API/synthesis", h.GetSynthesesAPI)
	authU.GET("/API/synthesis/:id", h.GetSynthesisAPI)
	authU.PUT("/API/synthesis/:id", h.UpdateSynthesisPurityAPI)
	authU.PUT("/API/synthesis/:id/form", h.FormSynthesisAPI)
	authM.PUT("/API/synthesis/:id/moderate", h.CompleteOrRejectSynthesisAPI)
	authU.DELETE("/API/synthesis", h.DeleteSynthesisAPI)

	authM.PUT("/API/synthesis/:id/update-result", h.UpdateSynthesisResultAPI)

	//домен м-м
	authU.DELETE("/API/reaction-synthesis", h.RemoveReactionFromSynthesisAPI)
	authU.PUT("/API/reaction-synthesis", h.UpdateReactionInSynthesisAPI)

	//домен пользователь
	router.POST("/API/users/register", h.RegisterUserAPI)
	authU.GET("/API/users/profile", h.GetUserProfileAPI) // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
	router.POST("/API/users/login", h.LoginUserAPI)
	authU.POST("/API/users/logout", h.LogoutUserAPI)
	authU.PUT("/API/users/profile", h.UpdateUserAPI)
}

func (h *Handler) RegisterStatic(router *gin.Engine) {
	router.LoadHTMLGlob("templates/*")
	router.Static("/static", "./resources")
}

func (h *Handler) errorHandler(ctx *gin.Context, errorStatusCode int, err error) {
	logrus.Error(err.Error())
	ctx.JSON(errorStatusCode, gin.H{
		"status":      "error",
		"description": err.Error(),
	})
}

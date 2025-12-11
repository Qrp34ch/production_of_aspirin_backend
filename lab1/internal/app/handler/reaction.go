package handler

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"lab1/internal/app/ds"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type SuccessResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func (h *Handler) GetReactions(ctx *gin.Context) {
	var reactions []ds.Reaction
	var err error

	searchQuery := ctx.Query("query")
	if searchQuery == "" {
		reactions, err = h.Repository.GetReactions()
		if err != nil {
			logrus.Error(err)
		}
	} else {
		reactions, err = h.Repository.GetReactionsByTitle(searchQuery)
		if err != nil {
			logrus.Error(err)
		}
	}

	userId, err := h.GetUserID(ctx)
	ctx.HTML(http.StatusOK, "index.html", gin.H{
		"synthesisCount": h.Repository.GetReactionsInSynthesis(userId),
		"reactions":      reactions,
		"id":             h.Repository.FindUserSynthesis(userId),
		"query":          searchQuery,
	})
}

func (h *Handler) GetReaction(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logrus.Error("Invalid ID format:", err)
		ctx.Redirect(http.StatusFound, "/reaction")
		return
	}

	reaction, err := h.Repository.GetReaction(id)
	if err != nil {
		logrus.Warnf("Reaction %d not found or deleted: %v", id, err)
		ctx.Redirect(http.StatusFound, "/reaction")
		return
	}

	ctx.HTML(http.StatusOK, "reaction.html", gin.H{
		"reaction": reaction,
	})
}

func (h *Handler) AddReactionInSynthesis(ctx *gin.Context) {
	strId := ctx.PostForm("reaction_id")
	id, err := strconv.Atoi(strId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	_, err = h.Repository.GetReaction(id)
	if err != nil {
		logrus.Warnf("Cannot add deleted reaction %d to synthesis", id)
		ctx.Redirect(http.StatusFound, "/reaction")
		return
	}

	userId, err := h.GetUserID(ctx)

	err = h.Repository.AddReactionInSynthesis(uint(id), userId)
	if err != nil && !strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
		return
	}
	ctx.Redirect(http.StatusFound, "/reaction")
}

func (h *Handler) GetSynthesis(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logrus.Error(err)
		ctx.Redirect(http.StatusFound, "/reaction")
		return
	}

	synthesisStatus, err := h.Repository.SynthesisStatusById(uint(id))
	if err != nil {
		logrus.Error(err)
		ctx.Redirect(http.StatusFound, "/reaction")
		return
	}

	if synthesisStatus == "удалён" {
		ctx.Redirect(http.StatusFound, "/reaction")
		return
	}

	synthesisReactions, err := h.Repository.GetSynthesisWithCounts(uint(id))
	if err != nil {
		logrus.Error(err)
		ctx.Redirect(http.StatusFound, "/reaction")
		return
	}
	userId, err := h.GetUserID(ctx)

	ctx.HTML(http.StatusOK, "synthesis.html", gin.H{
		"synthesisReactions": synthesisReactions,
		"id":                 id,
		"user":               h.Repository.GetUserNameByID(userId),
		"date":               h.Repository.GetDateUpdate(uint(id)),
		"purity":             h.Repository.GetPurity(uint(id)),
	})
}

func (h *Handler) RemoveSynthesis(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logrus.Error(err)
	}

	err = h.Repository.RemoveSynthesis(uint(id))
	ctx.Redirect(http.StatusFound, "/reaction")
}

// GetReactionsAPI godoc
// @Summary Get list of reactions
// @Description Get all reactions or search by title
// @Tags Reactions
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Success 200 {object} object{reactions=[]ds.Reaction,query=string}
// @Router /API/reaction [get]
func (h *Handler) GetReactionsAPI(ctx *gin.Context) {
	var reactions []ds.Reaction
	var err error

	searchQuery := ctx.Query("query")
	if searchQuery == "" {
		reactions, err = h.Repository.GetReactions()
		if err != nil {
			logrus.Error(err)
		}
	} else {
		reactions, err = h.Repository.GetReactionsByTitle(searchQuery)
		if err != nil {
			logrus.Error(err)
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"reactions": reactions,
		"query":     searchQuery,
	})
}

// GetReactionAPI godoc
// @Summary Get reaction by ID
// @Description Get specific reaction details
// @Tags Reactions
// @Accept json
// @Produce json
// @Param id path int true "Reaction ID"
// @Success 200 {object} object{reaction=ds.Reaction}
// @Failure 400 {object} object{status=string,description=string} "Bad Request"
// @Failure 404 {object} object{status=string,description=string} "Not Found"
// @Router /API/reaction/{id} [get]
func (h *Handler) GetReactionAPI(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)

	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	reaction, err := h.Repository.GetReaction(id)

	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Reaction not found or deleted"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"reaction": reaction,
	})
}

type ReactionInput struct {
	Title string `json:"title,omitempty"`
	//Src              string  json:"src,omitempty"
	//SrcUr            string  json:"src_ur,omitempty"
	Details          string  `json:"details,omitempty"`
	IsDelete         bool    `json:"is_delete,omitempty"`
	StartingMaterial string  `json:"starting_material,omitempty"`
	DensitySM        float32 `json:"density_sm,omitempty"`
	VolumeSM         float32 `json:"volume_sm,omitempty"`
	MolarMassSM      int     `json:"molar_mass_sm,omitempty"`
	ResultMaterial   string  `json:"result_material,omitempty"`
	DensityRM        float32 `json:"density_rm,omitempty"`
	VolumeRM         float32 `json:"volume_rm,omitempty"`
	MolarMassRM      int     `json:"molar_mass_rm,omitempty"`
}

// CreateReactionAPI godoc
// @Summary Create new reaction
// @Description Create a new chemical reaction
// @Tags Reactions
// @Accept json
// @Produce json
// @Param input body ReactionInput true "Reaction data"
// @Success 201 {object} object{status=string,description=string} "Created"
// @Failure 400 {object} object{status=string,description=string} "Bad Request"
// @Failure 500 {object} object{status=string,description=string} "Internal Server Error"
// @Router /API/create-reaction [post]
func (h *Handler) CreateReactionAPI(ctx *gin.Context) {
	var reactionInput struct {
		Title string `json:"title,omitempty"`
		//Src              string  json:"src,omitempty"
		//SrcUr            string  json:"src_ur,omitempty"
		Details          string  `json:"details,omitempty"`
		IsDelete         bool    `json:"is_delete,omitempty"`
		StartingMaterial string  `json:"starting_material,omitempty"`
		DensitySM        float32 `json:"density_sm,omitempty"`
		VolumeSM         float32 `json:"volume_sm,omitempty"`
		MolarMassSM      int     `json:"molar_mass_sm,omitempty"`
		ResultMaterial   string  `json:"result_material,omitempty"`
		DensityRM        float32 `json:"density_rm,omitempty"`
		VolumeRM         float32 `json:"volume_rm,omitempty"`
		MolarMassRM      int     `json:"molar_mass_rm,omitempty"`
	}

	if err := ctx.ShouldBindJSON(&reactionInput); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	newReaction := ds.Reaction{
		Title: reactionInput.Title,
		//Src              string  json:"src,omitempty"
		//SrcUr            string  json:"src_ur,omitempty"
		Details:          reactionInput.Details,
		IsDelete:         reactionInput.IsDelete,
		StartingMaterial: reactionInput.StartingMaterial,
		DensitySM:        reactionInput.DensitySM,
		//VolumeSM:         reactionInput.VolumeSM,
		MolarMassSM:    reactionInput.MolarMassSM,
		ResultMaterial: reactionInput.ResultMaterial,
		DensityRM:      reactionInput.DensityRM,
		//VolumeRM:         reactionInput.VolumeRM,
		MolarMassRM: reactionInput.MolarMassRM,
	}

	err := h.Repository.AddReaction(&newReaction)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"data":    newReaction,
		"message": "Реакция успешно создана",
	})
}

// ChangeReactionAPI godoc
// @Summary Update reaction
// @Description Update existing reaction
// @Tags Reactions
// @Accept json
// @Produce json
// @Param id path int true "Reaction ID"
// @Param input body ReactionInput true "Updated reaction data"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} object{status=string,description=string} "Bad Request"
// @Failure 500 {object} object{status=string,description=string} "Internal Server Error"
// @Router /API/reaction/{id} [put]
func (h *Handler) ChangeReactionAPI(ctx *gin.Context) {
	idReactionStr := ctx.Param("id")
	id, err := strconv.Atoi(idReactionStr)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	var reactionInput struct {
		Title            string  `json:"title,omitempty"`
		Src              string  `json:"src,omitempty"`
		SrcUr            string  `json:"src_ur,omitempty"`
		Details          string  `json:"details,omitempty"`
		IsDelete         bool    `json:"is_delete,omitempty"`
		StartingMaterial string  `json:"starting_material,omitempty"`
		DensitySM        float32 `json:"density_sm,omitempty"`
		VolumeSM         float32 `json:"volume_sm,omitempty"`
		MolarMassSM      int     `json:"molar_mass_sm,omitempty"`
		ResultMaterial   string  `json:"result_material,omitempty"`
		DensityRM        float32 `json:"density_rm,omitempty"`
		VolumeRM         float32 `json:"volume_rm,omitempty"`
		MolarMassRM      int     `json:"molar_mass_rm,omitempty"`
	}

	if err := ctx.ShouldBindJSON(&reactionInput); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	changeReaction := ds.Reaction{
		Title:            reactionInput.Title,
		Src:              reactionInput.Src,
		SrcUr:            reactionInput.SrcUr,
		Details:          reactionInput.Details,
		IsDelete:         reactionInput.IsDelete,
		StartingMaterial: reactionInput.StartingMaterial,
		DensitySM:        reactionInput.DensitySM,
		//VolumeSM:         reactionInput.VolumeSM,
		MolarMassSM:    reactionInput.MolarMassSM,
		ResultMaterial: reactionInput.ResultMaterial,
		DensityRM:      reactionInput.DensityRM,
		//VolumeRM:         reactionInput.VolumeRM,
		MolarMassRM: reactionInput.MolarMassRM,
	}
	err = h.Repository.ChangeReaction(uint(id), &changeReaction)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}
	updatedReaction, err := h.Repository.GetReaction(int(id))
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"data":    updatedReaction,
		"message": "Реакция успешно обновлена",
	})
}

// DeleteReactionAPI godoc
// @Summary Delete reaction
// @Description Delete reaction by ID (soft delete)
// @Tags Reactions
// @Accept json
// @Produce json
// @Param id path int true "Reaction ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} object{status=string,description=string} "Bad Request"
// @Failure 404 {object} object{status=string,description=string} "Not Found"
// @Failure 500 {object} object{status=string,description=string} "Internal Server Error"
// @Router /API/reaction/{id} [delete]
func (h *Handler) DeleteReactionAPI(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	_, err = h.Repository.GetReaction(int(id))
	if err != nil {
		h.errorHandler(ctx, http.StatusNotFound, err)
		return
	}

	err = h.Repository.DeleteReaction(uint(id))
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Реакция успешно удалена",
	})
}

// AddReactionInSynthesisAPI godoc
// @Summary Add reaction to synthesis
// @Description Add reaction to current user's synthesis
// @Tags Reactions
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Reaction ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} object{status=string,description=string} "Bad Request"
// @Failure 500 {object} object{status=string,description=string} "Internal Server Error"
// @Router /API/reaction/{id}/add-reaction-in-synthesis [post]
func (h *Handler) AddReactionInSynthesisAPI(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	userId, err := h.GetUserID(ctx)
	err = h.Repository.AddReactionInSynthesis(uint(id), userId)
	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Реакция добавлена в заявку",
	})
}

// UploadReactionImageAPI godoc
// @Summary Upload reaction image
// @Description Upload image for reaction
// @Tags Reactions
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "Reaction ID"
// @Param image formData file true "Image file"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} object{status=string,description=string} "Bad Request"
// @Failure 500 {object} object{status=string,description=string} "Internal Server Error"
// @Router /API/reaction/{id}/image [post]
func (h *Handler) UploadReactionImageAPI(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	file, err := ctx.FormFile("image")
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, fmt.Errorf("файл изображения обязателен"))
		return
	}

	err = h.Repository.UploadReactionImage(uint(id), file)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	updatedReaction, err := h.Repository.GetReaction(int(id))
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"data":    updatedReaction,
		"message": "Изображение успешно загружено",
	})
}

// GetSynthesisIconAPI godoc
// @Summary Get synthesis icon data
// @Description Get current user's synthesis ID and items count
// @Tags Syntheses
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} SuccessResponse
// @Router /API/synthesis/icon [get]
func (h *Handler) GetSynthesisIconAPI(ctx *gin.Context) {
	userID, err := h.GetUserID(ctx)
	if err != nil {
		fmt.Printf("nuiladno")
	}

	synthesisID := h.Repository.GetSynthesisID(userID)
	synthesisCount := h.Repository.GetSynthesisCount(userID)

	ctx.JSON(http.StatusOK, gin.H{
		"status":       "success",
		"id_synthesis": synthesisID,
		"items_count":  synthesisCount,
	})
}

// GetSynthesesAPI godoc
// @Summary Get list of syntheses
// @Description Get syntheses with optional filtering
// @Tags Syntheses
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param status query string false "Status filter"
// @Param start_date query string false "Start date (YYYY-MM-DD)"
// @Param end_date query string false "End date (YYYY-MM-DD)"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} object{status=string,description=string} "Bad Request"
// @Failure 500 {object} object{status=string,description=string} "Internal Server Error"
// @Router /API/synthesis [get]
func (h *Handler) GetSynthesesAPI(ctx *gin.Context) {
	var filter struct {
		Status    string `form:"status"`
		StartDate string `form:"start_date"`
		EndDate   string `form:"end_date"`
	}

	type SynthesisWithLogin struct {
		ID             uint    `form:"id"`
		Status         string  `form:"status"`
		DateCreate     string  `form:"date_create"`
		DateUpdate     string  `form:"date_update"`
		DateFinish     string  `form:"date_finish"`
		CreatorID      uint    `form:"creator_id"`
		ModeratorID    uint    `form:"moderator_id"`
		Purity         float32 `form:"purity"`
		CreatorLogin   string  `form:"creator_login"`
		ModeratorLogin string  `form:"moderator_login"`
	}

	if err := ctx.ShouldBindQuery(&filter); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	userID, err := h.GetUserID(ctx)

	syntheses, err := h.Repository.GetSyntheses(filter.Status, filter.StartDate, filter.EndDate, userID)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	response := make([]SynthesisWithLogin, len(syntheses))
	for i, calc := range syntheses {
		response[i] = SynthesisWithLogin{
			ID:           calc.ID,
			Status:       calc.Status,
			DateCreate:   calc.DateCreate.Format("02.01.2006"),
			DateUpdate:   calc.DateUpdate.Format("02.01.2006"),
			CreatorLogin: calc.Creator.Login,
			Purity:       calc.Purity,
		}

		if calc.DateFinish.Valid {
			response[i].DateFinish = calc.DateFinish.Time.Format("02.01.2006")
		}

		if calc.Moderator.ID != 0 {
			response[i].ModeratorLogin = calc.Moderator.Login
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   response,
		"count":  len(response),
	})
}

// GetSynthesisAPI godoc
// @Summary Get synthesis by ID
// @Description Get detailed information about synthesis
// @Tags Syntheses
// @Accept json
// @Produce json
// @Param id path int true "Synthesis ID"
// @Success 200 {object} object{status=string,data=object}
// @Failure 400 {object} object{status=string,description=string} "Bad Request"
// @Failure 404 {object} object{status=string,description=string} "Not Found"
// @Router /API/synthesis/{id} [get]
func (h *Handler) GetSynthesisAPI(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	synthesis, reactions, err := h.Repository.GetSynthesisByID(uint(id))
	if err != nil {
		h.errorHandler(ctx, http.StatusNotFound, err)
		return
	}
	synthesisReactions, err := h.Repository.GetSynthesisWithCounts(uint(id))
	if err != nil {
		h.errorHandler(ctx, http.StatusNotFound, err)
		return
	}

	synthesisFull := struct {
		ID             uint
		Status         string
		DateCreate     string
		DateUpdate     string
		DateFinish     string
		CreatorLogin   string
		ModeratorLogin string
		Purity         float32
		Reactions      []ds.Reaction
	}{
		ID:           synthesis.ID,
		Status:       synthesis.Status,
		DateCreate:   synthesis.DateCreate.Format("02.01.2006"),
		DateUpdate:   synthesis.DateUpdate.Format("02.01.2006"),
		CreatorLogin: synthesis.Creator.Login,
		Purity:       synthesis.Purity,
		Reactions:    make([]ds.Reaction, len(reactions)), // используем отдельно загруженные fuels
	}

	if synthesis.DateFinish.Valid {
		synthesisFull.DateFinish = synthesis.DateFinish.Time.Format("02.01.2006")
	}

	if synthesis.Moderator.ID != 0 {
		synthesisFull.ModeratorLogin = synthesis.Moderator.Login
	}

	for i, reaction := range reactions {
		synthesisFull.Reactions[i] = ds.Reaction{
			ID:               reaction.ID,
			Title:            reaction.Title,
			Src:              reaction.Src,
			SrcUr:            reaction.SrcUr,
			Details:          reaction.Details,
			IsDelete:         reaction.IsDelete,
			StartingMaterial: reaction.StartingMaterial,
			DensitySM:        reaction.DensitySM,
			//VolumeSM:         reaction.VolumeSM,
			MolarMassSM:    reaction.MolarMassSM,
			ResultMaterial: reaction.ResultMaterial,
			DensityRM:      reaction.DensityRM,
			//VolumeRM:         reaction.VolumeRM,
			MolarMassRM: reaction.MolarMassRM,
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"data":    synthesisFull,
		"data_pr": synthesisReactions,
	})
}

type InputPurity struct {
	Purity float64 `json:"purity" binding:"required"`
}

// UpdateSynthesisPurityAPI godoc
// @Summary Update synthesis purity
// @Description Update purity percentage for synthesis
// @Tags Syntheses
// @Accept json
// @Produce json
// @Param id path int true "Synthesis ID"
// @Param input body InputPurity true "Purity data"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} object{status=string,description=string} "Bad Request"
// @Router /API/synthesis/{id} [put]
func (h *Handler) UpdateSynthesisPurityAPI(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	var input struct {
		Purity float64 `json:"purity" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&input); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	err = h.Repository.UpdateSynthesisPurity(uint(id), input.Purity)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	updatedSynthesis, _, err := h.Repository.GetSynthesisByID(uint(id))
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"data":    updatedSynthesis,
		"message": "концентрация успешно обновлена",
	})
}

// FormSynthesisAPI godoc
// @Summary Form synthesis
// @Description Change synthesis status to "сформирован"
// @Tags Syntheses
// @Accept json
// @Produce json
// @Param id path int true "Synthesis ID"
// @Success 200 {object} object{status=string,data=object,reactions=[]ds.Reaction,message=string}
// @Failure 400 {object} object{status=string,description=string} "Bad Request"
// @Failure 500 {object} object{status=string,description=string} "Internal Server Error"
// @Router /API/synthesis/{id}/form [put]
func (h *Handler) FormSynthesisAPI(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	err = h.Repository.FormSynthesis(uint(id))
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	updatedSynthesis, reactions, err := h.Repository.GetSynthesisByID(uint(id))
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":    "success",
		"data":      updatedSynthesis,
		"reactions": reactions,
		"message":   "Синтез успешно сформирован",
	})
}

type CompleteOrRejectRequest struct {
	NewStatus bool `json:"new_status" binding:"required"`
}

// CompleteOrRejectSynthesisAPI godoc
// @Summary Complete or reject synthesis
// @Description Complete or reject synthesis by moderator
// @Tags Syntheses
// @Accept json
// @Produce json
// @Param id path int true "Synthesis ID"
// @Param input body CompleteOrRejectRequest true "Action data"
// @Success 200 {object} object{status=string,data=object,reactions=[]ds.Reaction,message=string}
// @Failure 400 {object} object{status=string,description=string} "Bad Request"
// @Router /API/synthesis/{id}/moderate [put]
func (h *Handler) CompleteOrRejectSynthesisAPI(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	var input struct {
		NewStatus bool `json:"new_status"`
	}

	if err := ctx.ShouldBindJSON(&input); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	moderatorID := uint(2)

	err = h.Repository.CompleteOrRejectSynthesis(uint(id), moderatorID, input.NewStatus)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	updatedSynthesis, reactions, err := h.Repository.GetSynthesisByID(uint(id))
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	message := "Заявка отклонена"
	if input.NewStatus {
		message = "Заявка завершена"
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":    "success",
		"data":      updatedSynthesis,
		"reactions": reactions,
		"message":   message,
	})
}

// DeleteSynthesisAPI godoc
// @Summary Delete current user's synthesis
// @Description Delete current user's active synthesis
// @Tags Syntheses
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} SuccessResponse
// @Failure 500 {object} object{status=string,description=string} "Internal Server Error"
// @Router /API/synthesis [delete]
func (h *Handler) DeleteSynthesisAPI(ctx *gin.Context) {
	userId, err := h.GetUserID(ctx)
	id := h.Repository.GetSynthesisID(userId)

	err = h.Repository.DeleteSynthesis(uint(id))
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Синтез успешно удален",
	})
}

// RemoveReactionFromSynthesisAPI godoc
// @Summary Remove reaction from synthesis
// @Description Remove reaction from current user's synthesis
// @Tags Synthesis-Reactions
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param reaction_id query int true "Reaction ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} object{status=string,description=string} "Bad Request"
// @Router /API/reaction-synthesis [delete]
func (h *Handler) RemoveReactionFromSynthesisAPI(ctx *gin.Context) {
	userId, err := h.GetUserID(ctx)
	synthesisID := h.Repository.GetSynthesisID(userId)
	reactionIDStr := ctx.Query("reaction_id")
	reactionID, err := strconv.Atoi(reactionIDStr)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	err = h.Repository.RemoveReactionFromSynthesis(uint(synthesisID), uint(reactionID))
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Реакция удалена из синтеза",
	})
}

type UpdateReactionInSynthesisRequest struct {
	ReactionID uint    `json:"reaction_id" binding:"required"`
	VolumeSM   float64 `json:"volume_sm" binding:"required"`
}

// UpdateReactionInSynthesisAPI godoc
// @Summary Update reaction in synthesis
// @Description Update reaction volume in current user's synthesis
// @Tags Synthesis-Reactions
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body UpdateReactionInSynthesisRequest true "Update data"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} object{status=string,description=string} "Bad Request"
// @Router /API/reaction-synthesis [put]
func (h *Handler) UpdateReactionInSynthesisAPI(ctx *gin.Context) {
	userId, err := h.GetUserID(ctx)
	synthesisID := h.Repository.GetSynthesisID(userId)

	var input struct {
		ReactionID uint    `json:"reaction_id" binding:"required"`
		VolumeSM   float64 `json:"volume_sm" binding:"required"`
	}
	//var err error
	if err = ctx.ShouldBindJSON(&input); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	err = h.Repository.UpdateReactionInSynthesis(uint(synthesisID), input.ReactionID, input.VolumeSM)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Данные реакции обновлены в синтезе",
	})
}

type RegisterReq struct {
	Login string `json:"login"` // лучше назвать то же самое что login
	Pass  string `json:"pass"`
	FIO   string `json:"fio"`
}

type RegisterResp struct {
	Ok bool `json:"ok"`
}

// RegisterUserAPI godoc
// @Summary Register new user
// @Description Register new user in the system
// @Tags Users
// @Accept json
// @Produce json
// @Param input body RegisterReq true "User registration data"
// @Success 200 {object} RegisterResp
// @Failure 400 {object} object{status=string,description=string} "Bad Request"
// @Failure 500 {object} object{status=string,description=string} "Internal Server Error"
// @Router /API/users/register [post]
func (h *Handler) RegisterUserAPI(gCtx *gin.Context) {
	req := &RegisterReq{}

	err := json.NewDecoder(gCtx.Request.Body).Decode(req)
	if err != nil {
		gCtx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if req.Pass == "" {
		gCtx.AbortWithError(http.StatusBadRequest, fmt.Errorf("pass is empty"))
		return
	}

	if req.Login == "" {
		gCtx.AbortWithError(http.StatusBadRequest, fmt.Errorf("name is empty"))
		return
	}

	err = h.Repository.RegisterUser(req.Login, generateHashString(req.Pass), req.FIO)
	if err != nil {
		gCtx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	gCtx.JSON(http.StatusOK, &RegisterResp{
		Ok: true,
	})
}

func generateHashString(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// GetUserProfileAPI godoc
// @Summary Get user profile
// @Description Get current user's profile information
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} object{status=string,data=ds.Users}
// @Failure 400 {object} object{status=string,description=string} "Bad Request"
// @Router /API/users/profile [get]
func (h *Handler) GetUserProfileAPI(ctx *gin.Context) {
	userID, err := h.GetUserID(ctx)

	user, err := h.Repository.GetUserProfile(userID)
	if err != nil {
		h.errorHandler(ctx, http.StatusNotFound, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   user,
	})
}

type LoginReq struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type LoginResp struct {
	ExpiresIn   int       `json:"expires_in"`
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	User        *UserInfo `json:"user"`
}

type UserInfo struct {
	ID          uint   `json:"id" example:"1"`
	Login       string `json:"login" example:"admin"`
	FIO         string `json:"fio" example:"Иванов Иван Иванович"`
	IsModerator bool   `json:"is_moderator" example:"true"`
}

// LoginUserAPI godoc
// @Summary User login
// @Description Authenticate user and return JWT token
// @Tags Users
// @Accept json
// @Produce json
// @Param input body LoginReq true "Login credentials"
// @Success 200 {object} LoginResp
// @Failure 400 {object} object{status=string,description=string} "Bad Request"
// @Failure 403 {object} object{status=string,description=string} "Forbidden"
// @Failure 500 {object} object{status=string,description=string} "Internal Server Error"
// @Router /API/users/login [post]
func (h *Handler) LoginUserAPI(ctx *gin.Context) {
	req := &LoginReq{}

	err := json.NewDecoder(ctx.Request.Body).Decode(req)
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	// Аутентифицируем пользователя
	user, err := h.Repository.AuthUser(req.Login, generateHashString(req.Password))
	fmt.Printf(generateHashString(req.Password))
	if err != nil {
		ctx.AbortWithStatus(http.StatusForbidden)
		return
	}

	cfg := h.Config

	// Генерируем JWT токен
	token := jwt.NewWithClaims(cfg.JWT.SigningMethod, &ds.JWTClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(cfg.JWT.ExpiresIn).Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    "bitop-admin",
			Subject:   strconv.FormatUint(uint64(user.ID), 10), // добавляем ID пользователя
		},
		UserUUID: uuid.New(),
		IsAdmin:  user.IsModerator,
		UserID:   user.ID,
		Scopes:   []string{},
	})

	strToken, err := token.SignedString([]byte(cfg.JWT.Token))
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, fmt.Errorf("cant create str token"))
		return
	}

	ctx.JSON(http.StatusOK, LoginResp{
		ExpiresIn:   int(cfg.JWT.ExpiresIn.Seconds()), // конвертируем в секунды
		AccessToken: strToken,
		TokenType:   "Bearer",
		User: &UserInfo{
			ID:          user.ID,
			Login:       user.Login,
			FIO:         user.FIO,
			IsModerator: user.IsModerator,
		},
	})
}

// LogoutUserAPI godoc
// @Summary User logout
// @Description Logout user and invalidate token
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} object{message=string}
// @Router /API/users/logout [post]
func (h *Handler) LogoutUserAPI(ctx *gin.Context) {
	tokenString := ctx.GetHeader("Authorization")
	if tokenString == "" {
		ctx.JSON(http.StatusOK, gin.H{
			"message": "Выход выполнен",
		})
		return
	}

	if !strings.HasPrefix(tokenString, jwtPrefix) {
		ctx.JSON(http.StatusOK, gin.H{
			"message": "Выход выполнен",
		})
		return
	}

	// Отрезаем префикс
	tokenString = tokenString[len(jwtPrefix):]

	// Парсим токен чтобы получить expiration time
	claims := &ds.JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.Config.JWT.Token), nil
	})

	if err != nil || !token.Valid {
		// Если токен невалиден, все равно считаем выход успешным
		ctx.JSON(http.StatusOK, gin.H{
			"message": "Выход выполнен",
		})
		return
	}

	// Добавляем токен в черный список
	if h.Repository.RedisClient != nil {
		// Время жизни в черном списке = оставшееся время жизни токена
		remainingTTL := time.Unix(claims.ExpiresAt, 0).Sub(time.Now())
		if remainingTTL > 0 {
			err = h.Repository.RedisClient.WriteJWTToBlacklist(ctx.Request.Context(), tokenString, remainingTTL)
			if err != nil {
				// Логируем ошибку, но все равно возвращаем успех
				logrus.Errorf("Ошибка добавления токена в черный список: %v", err)
			}
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Выход выполнен успешно",
	})
}

type UpdateUserRequest struct {
	Login    *string `json:"login,omitempty"`
	Name     *string `json:"name,omitempty"`
	Password *string `json:"password,omitempty"`
}

// UpdateUserAPI godoc
// @Summary Update user profile
// @Description Update current user's profile information
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body UpdateUserRequest true "User update data"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} object{status=string,description=string} "Bad Request"
// @Router /API/users/profile [put]
func (h *Handler) UpdateUserAPI(ctx *gin.Context) {
	userID, err := h.GetUserID(ctx)

	var input struct {
		Login    *string `json:"login,omitempty"`
		Name     *string `json:"name,omitempty"`
		Password *string `json:"password,omitempty"`
	}
	if err := ctx.ShouldBindJSON(&input); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	updates := make(map[string]interface{})
	if input.Login != nil {
		updates["login"] = *input.Login
	}
	if input.Name != nil {
		updates["name"] = *input.Name
	}

	//if input.Password != nil {
	//	input.Password = generateHashString(input.Password)
	//	updates["password"] = *input.Password
	//}
	if len(updates) == 0 {
		h.errorHandler(ctx, http.StatusBadRequest, fmt.Errorf("нет полей для обновления"))
		return
	}
	user, err := h.Repository.UpdateUser(userID, updates)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"data":    user,
		"message": "Данные обновлены",
	})
}

// GetUserID получает userID из контекста (установленного middleware)
func (h *Handler) GetUserID(ctx *gin.Context) (uint, error) {
	userID, exists := ctx.Get("userID")
	if !exists {
		return 0, fmt.Errorf("требуется авторизация")
	}

	return userID.(uint), nil
}

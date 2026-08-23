package handlers

import (
	"context"
	"net/http"
	"time"

	"streamly/internal/database"
	"streamly/internal/middleware"
	"streamly/internal/models"
	"streamly/internal/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AdminHandler struct {

	auth *services.AuthService
	db *database.DB

}

func NewAdminHandler(auth *services.AuthService, db *database.DB) *AdminHandler {

	return &AdminHandler{auth: auth, db: db}

}

type createCodeRequest struct {

	MaxUses int `json:"maxUses"`
	ExpiresIn string `json:"expiresIn"`

}

func (h *AdminHandler) CreateAccessCode(c *gin.Context) {

	var req createCodeRequest

	if err := c.ShouldBindJSON(&req); err != nil {

		writeError(c, http.StatusBadRequest, "invalid request")
		return

	}

	var expiresAt *time.Time

	if req.ExpiresIn != "" {

		d, err := time.ParseDuration(req.ExpiresIn)

		if err != nil {

			writeError(c, http.StatusBadRequest, "invalid expiresIn duration")
			return

		}

		t := time.Now().Add(d)
		expiresAt = &t

	}

	userID := c.GetString(middleware.UserIDKey)

	code, err := h.auth.CreateAccessCode(c.Request.Context(), userID, req.MaxUses, expiresAt)

	if err != nil {

		handleServiceError(c, err)
		return

	}

	c.JSON(http.StatusCreated, code)

}

func (h *AdminHandler) ListAccessCodes(c *gin.Context) {

	codes, err := h.auth.ListAccessCodes(c.Request.Context())

	if err != nil {

		handleServiceError(c, err)
		return

	}

	if codes == nil {

		codes = []models.AccessCode{}

	}

	c.JSON(http.StatusOK, codes)

}

func (h *AdminHandler) DeleteAccessCode(c *gin.Context) {

	if err := h.auth.DeleteAccessCode(c.Request.Context(), c.Param("code")); err != nil {

		handleServiceError(c, err)
		return

	}

	c.Status(http.StatusNoContent)

}

func (h *AdminHandler) GetServiceInterruption(c *gin.Context) {

	var interruption models.ServiceInterruption

	err := h.db.ServiceInterruption().FindOne(c.Request.Context(), bson.M{}).Decode(&interruption)

	if err == mongo.ErrNoDocuments {

		c.JSON(http.StatusOK, models.ServiceInterruption{Enabled: false})
		return

	}

	if err != nil {

		writeError(c, http.StatusInternalServerError, "failed to load service interruption")
		return

	}

	c.JSON(http.StatusOK, interruption)

}

type updateServiceInterruptionRequest struct {

	Enabled bool   `json:"enabled"`
	Title   string `json:"title"`
	Message string `json:"message"`

}

func (h *AdminHandler) UpdateServiceInterruption(c *gin.Context) {

	var req updateServiceInterruptionRequest

	if err := c.ShouldBindJSON(&req); err != nil {

		writeError(c, http.StatusBadRequest, "invalid request")
		return

	}

	now := time.Now()

	update := bson.M{

		"$set": bson.M{

			"enabled":   req.Enabled,
			"title":     req.Title,
			"message":   req.Message,
			"updatedAt": now,

		},

	}

	opts := options.Update().SetUpsert(true)

	if _, err := h.db.ServiceInterruption().UpdateOne(c.Request.Context(), bson.M{}, update, opts); err != nil {

		writeError(c, http.StatusInternalServerError, "failed to save service interruption")
		return

	}

	var saved models.ServiceInterruption

	_ = h.db.ServiceInterruption().FindOne(context.Background(), bson.M{}).Decode(&saved)

	c.JSON(http.StatusOK, saved)

}

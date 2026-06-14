package handlers

import (
	"net/http"
	"time"

	"streamly/internal/middleware"
	"streamly/internal/models"
	"streamly/internal/services"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {

	auth *services.AuthService

}

func NewAdminHandler(auth *services.AuthService) *AdminHandler {

	return &AdminHandler{auth: auth}

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

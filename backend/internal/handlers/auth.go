package handlers

import (
	"net/http"
	"time"

	"streamly/internal/middleware"
	"streamly/internal/services"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	auth *services.AuthService
}

func NewAuthHandler(auth *services.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type registerRequest struct {
	Email      string `json:"email" binding:"required"`
	Password   string `json:"password" binding:"required"`
	AccessCode string `json:"accessCode" binding:"required"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request")
		return
	}

	user, token, err := h.auth.Register(c.Request.Context(), req.Email, req.Password, req.AccessCode)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	h.setTokenCookie(c, token)
	c.JSON(http.StatusCreated, gin.H{
		"id":      user.ID.Hex(),
		"email":   user.Email,
		"isAdmin": user.IsAdmin,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request")
		return
	}

	user, token, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	h.setTokenCookie(c, token)
	c.JSON(http.StatusOK, gin.H{
		"id":      user.ID.Hex(),
		"email":   user.Email,
		"isAdmin": user.IsAdmin,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	secure, domain := h.auth.CookieSettings()
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("streamly_token", "", -1, "/", domain, secure, true)
	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.GetString(middleware.UserIDKey)
	user, err := h.auth.GetUser(c.Request.Context(), userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      user.ID.Hex(),
		"email":   user.Email,
		"isAdmin": user.IsAdmin,
	})
}

func (h *AuthHandler) setTokenCookie(c *gin.Context, token string) {
	secure, domain := h.auth.CookieSettings()
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("streamly_token", token, int((7 * 24 * time.Hour).Seconds()), "/", domain, secure, true)
}

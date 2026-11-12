package handlers

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/plannedwallet/internal/captcha"
	"github.com/plannedwallet/internal/trace"
)

// GetCaptcha handles GET /api/v1/captcha
func (h *Handler) GetCaptcha(c *gin.Context) {
	trace.LogInfo(c.Request.Context(), "Generating new CAPTCHA challenge")
	res, err := captcha.Generate()
	if err != nil {
		h.sendError(c, http.StatusInternalServerError, "CAPTCHA_GENERATION_FAILED", "Failed to generate CAPTCHA challenge.")
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetCaptchaImage handles GET /api/v1/captcha/:captchaId (and :captchaId.png)
func (h *Handler) GetCaptchaImage(c *gin.Context) {
	captchaID := strings.TrimSuffix(c.Param("captchaId"), ".png")
	trace.LogInfo(c.Request.Context(), "Serving CAPTCHA image for challenge: %s", captchaID)
	c.Header("Content-Type", "image/png")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	var buf bytes.Buffer
	if err := captcha.WriteImage(&buf, captchaID); err != nil {
		h.sendError(c, http.StatusNotFound, "CAPTCHA_NOT_FOUND", "CAPTCHA challenge not found or expired.")
		return
	}
	c.Data(http.StatusOK, "image/png", buf.Bytes())
}

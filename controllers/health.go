package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthController struct{}

// &@BasePath	/

// Health Check
//
//	@Summary	Health Check
//	@Schemes
//	@Description	Health check
//	@Tags			status
//	@Accept			json
//	@Produce		plain
//	@Success		200	{string}	Working!
//	@Router			/health [get]
func (h HealthController) Status(c *gin.Context) {
	c.String(http.StatusOK, "Working!")
}

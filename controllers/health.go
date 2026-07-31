package controllers

import (
	"net/http"
	"time"

	"github.com/brian-l-johnson/CashierStatusBoard/v2/models"
	"github.com/gin-gonic/gin"
)

type HealthController struct{}

var startTime = time.Now()

// HealthResponse represents the health check response
type HealthResponse struct {
	Status           string `json:"status"`
	Mode             string `json:"mode"`
	UptimeSeconds    int64  `json:"uptime_seconds"`
	Timestamp        string `json:"timestamp"`
	CashierCount     int64  `json:"cashier_count"`
	ActiveSSEClients int    `json:"active_sse_clients"`
}

// &@BasePath	/

// Health Check
//
//	@Summary	Health Check
//	@Schemes
//	@Description	Health check with system status
//	@Tags			status
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	HealthResponse
//	@Router			/health [get]
func (h HealthController) Status(c *gin.Context) {
	// Get database instance
	db := models.GetDB()

	// Count cashiers
	var cashierCount int64
	db.Model(&models.Cashier{}).Count(&cashierCount)

	// clients is shared with the SSE handlers in cashiers.go, which mutate it
	// as displays connect and disconnect. Reading a map while another
	// goroutine writes it is a fatal runtime error in Go -- not a recoverable
	// panic -- so gin.Recovery() would not save us here.
	mutex.Lock()
	activeClients := len(clients)
	mutex.Unlock()

	// Build response
	response := HealthResponse{
		Status:           "healthy",
		Mode:             "server",
		UptimeSeconds:    int64(time.Since(startTime).Seconds()),
		Timestamp:        time.Now().UTC().Format(time.RFC3339),
		CashierCount:     cashierCount,
		ActiveSSEClients: activeClients,
	}

	c.JSON(http.StatusOK, response)
}

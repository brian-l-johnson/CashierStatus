package controllers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/brian-l-johnson/CashierStatusBoard/v2/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CashierController struct{}

var mutex = &sync.Mutex{}

type Message struct {
	Type    string `json:"type"`
	Cashier string `json:"cashier"`
	Serving string `json:"serving"`
}

// clients holds one buffered channel per connected SSE client. The broadcaster
// sends each update to every channel.
var clients = make(map[chan Message]bool)

// &@BasePath	/

// Get Cashiers
//
//	@Summary	Get Cashiers
//	@Schemes
//	@Description	Get Cashiers
//	@Tags			Cashiers
//	@Accept			json
//	@Produce		plain
//	@Success		200	{string}	Working!
//	@Router			/cashiers [get]
func (h CashierController) GetCashiers(c *gin.Context) {
	db := models.GetDB()
	var cashiers []models.Cashier

	result := db.Find(&cashiers)
	if result.Error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": result.Error})
		return
	}
	c.IndentedJSON(http.StatusOK, cashiers)
}

// create cashier godoc
//
// @Summary Create Cashier
// @Description Create a new Cashier
// @Tags Cashiers
// @Accept json
// @Produces json
// @Param create body models.Cashier true "cashier data"
// @Success 200 {string} result
// @Router /cashiers [post]
func (h CashierController) CreateCashier(c *gin.Context) {
	cashierreq := new(models.Cashier)
	if err := c.BindJSON(&cashierreq); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"status": "error", "message": "failed to bind request"})
		return
	}
	cashierdb := new(models.Cashier)
	db := models.GetDB()
	result := db.First(&cashierdb, "id=?", cashierreq.ID)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			cr := db.Create(&cashierreq)
			if cr.Error != nil {
				c.IndentedJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": cr.Error})
				return
			} else {
				c.IndentedJSON(http.StatusOK, gin.H{"status": "success", "message": "cashier created"})
				return
			}
		}
	} else {
		c.IndentedJSON(http.StatusOK, gin.H{"staus": "error", "message": "cashier already exists"})
	}
}

// update cashier order godoc
//
// @Summary set the order number a cashier is servicing
// @Tags Cashiers
// @Accept json
// @Produces json
// @Param cid	path	string			true "Cashier ID"
// @Param order	body	models.OrderReq	true "Order Info"
// @Success 200 {string} result
// @Router /cashiers/{cid} [patch]
func (h CashierController) UpdateCashier(c *gin.Context) {
	db := models.GetDB()
	var cashier models.Cashier
	result := db.First(&cashier, "ID=?", c.Param("cid"))
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.IndentedJSON(http.StatusOK, gin.H{"status": "error", "message": "cashier not found"})
			return
		} else {
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": result.Error})
			return
		}
	} else {
		var orderReq models.OrderReq
		if err := c.BindJSON(&orderReq); err != nil {
			return
		}
		cashier.Serving = orderReq.OrderNum
		result := db.Save(&cashier)
		if result.Error != nil {
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": result.Error})
			return
		} else {
			msg := Message{Type: "update", Cashier: strconv.FormatUint(uint64(cashier.ID), 10), Serving: cashier.Serving}

			mutex.Lock()
			for client := range clients {
				// Non-blocking send: skip clients whose buffer is full so a
				// slow consumer can't stall the broadcast.
				select {
				case client <- msg:
				default:
					fmt.Println("dropping update for slow SSE client")
				}
			}
			mutex.Unlock()
			c.IndentedJSON(http.StatusOK, gin.H{"status": "success", "message": "cashier updates"})
		}
	}
}

// delete cashier godoc
//
// @Summary delete cashier
// @Tags Cashiers
// @Accept json
// @Produces json
// @Param cid path string true "Cashier ID"
// @Success 200 {string} result
// @Router /cashiers/{cid} [delete]
func (h CashierController) DeleteCashier(c *gin.Context) {
	db := models.GetDB()
	var cashier models.Cashier
	result := db.First(&cashier, "ID=?", c.Param("tid"))
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.IndentedJSON(http.StatusOK, gin.H{"status": "error", "message": "cashier not found"})
			return
		} else {
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": result.Error})
			return
		}
	} else {
		result := db.Delete(&cashier)
		if result.Error != nil {
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": result.Error})
			return
		} else {
			c.IndentedJSON(http.StatusOK, gin.H{"status": "success", "message": "cashier deleted"})
		}
	}

}
func (h CashierController) GetCashierUpdates(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	messageChan := make(chan Message, 16)
	mutex.Lock()
	clients[messageChan] = true
	mutex.Unlock()

	defer func() {
		mutex.Lock()
		delete(clients, messageChan)
		mutex.Unlock()
		close(messageChan)
	}()

	// Heartbeat keeps the connection alive through idle-timeout proxies and
	// lets a dead connection surface quickly. SSE comments (": ...") are
	// ignored by EventSource, so no client handling is needed.
	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()

	c.Stream(func(w io.Writer) bool {
		select {
		case msg, ok := <-messageChan:
			if !ok {
				return false
			}
			c.SSEvent("message", msg)
			return true
		case <-heartbeat.C:
			fmt.Fprint(w, ": ping\n\n")
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}

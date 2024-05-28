package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/brian-l-johnson/CashierStatusBoard/v2/models"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

type CashierController struct{}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var mutex = &sync.Mutex{}

type Message struct {
	Type    string `json:"type"`
	Cashier string `json:"cashier"`
	Serving string `json:"serving"`
}

var clients = make(map[*websocket.Conn]bool)

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

			for client := range clients {
				err := client.WriteJSON(msg)
				if err != nil {
					fmt.Println(err)
					mutex.Lock()
					client.Close()
					delete(clients, client)
					mutex.Unlock()
				}
			}
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
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "Error", "message": err.Error})
		return
	} else {
		defer conn.Close()
		mutex.Lock()
		clients[conn] = true
		mutex.Unlock()
		for {
			var msg string
			err := conn.ReadJSON(msg)
			if err != nil {
				fmt.Println(err)
				mutex.Lock()
				delete(clients, conn)
				mutex.Unlock()
				return
			}
			fmt.Println("got message from clinet:" + msg)
		}
	}
}

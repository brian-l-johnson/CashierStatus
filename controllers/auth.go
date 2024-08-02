package controllers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"

	"github.com/brian-l-johnson/CashierStatusBoard/v2/models"
	"github.com/gin-gonic/gin"
)

type AuthController struct{}

// &@BasePath	/

// MAC Message
//
//	@Summary	Calculate HMAC for message
//	@Schemes
//	@Description	Calculate HMAC
//	@Tags			Auth
//	@Accept			json
//	@Produce		plain
//	@Param create body models.MacReq true "cashier data"
//	@Success		200	{string}	Working!
//	@Router			/auth/mac [post]
func (h AuthController) Mac(c *gin.Context) {
	macreq := new(models.MacReq)
	if err := c.BindJSON(&macreq); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"status": "error", "message": "failed to bind request"})
		return
	}
	mac := hmac.New(sha256.New, []byte("badbadbad"))
	v := macreq.Action + ":" + macreq.Value
	mac.Write([]byte(v))
	mv := mac.Sum(nil)
	c.IndentedJSON(http.StatusOK, gin.H{"status": "sucess", "hmac": mv})
}

// Verify MAC
//
// @Summary	Verify MAC for message
//
//	@Schemes
//	@Description	Verify HMAC
//	@Tags			Auth
//	@Accept			json
//	@Produce		plain
//	@Param create body models.VerifyMacReq true "data"
//	@Success		200	{string}	Working!
//	@Router			/auth/verify [post]
func (h AuthController) Verify(c *gin.Context) {
	verifyreq := new(models.VerifyMacReq)
	if err := c.BindJSON(&verifyreq); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"status": "error", "message": "failed to bind request"})
		return
	}
	mac := hmac.New(sha256.New, []byte("badbadbad"))
	v := verifyreq.Action + ":" + verifyreq.Value
	mac.Write([]byte(v))
	mv := mac.Sum(nil)
	//fmt.Println("comparing" + string(mv) + " to " + verifyreq.Mac)
	dmv, err := base64.StdEncoding.DecodeString(verifyreq.Mac)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"status": "error", "message": "mac not base64 encoded"})
	}
	if hmac.Equal(mv, dmv) {
		c.IndentedJSON(http.StatusOK, gin.H{"status": "succes", "result": true})
	} else {
		c.IndentedJSON(http.StatusOK, gin.H{"status": "succes", "result": false})
	}

}

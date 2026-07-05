package controllers

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/brian-l-johnson/CashierStatusBoard/v2/models"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthController struct{}

// stationRE matches merch-app's alphanum-max-20 station ID rule.
var stationRE = regexp.MustCompile(`^[a-zA-Z0-9]{1,20}$`)

// &@BasePath	/

// Login godoc
//
//	@Summary		Login
//	@Description	Login a user
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Param			login	body		models.LoginReq	true	"Login Data"
//	@Success		200		{string}	result
//	@Router			/auth/login [post]
func (a AuthController) Login(c *gin.Context) {
	var lr = new(models.LoginReq)
	if err := c.BindJSON(&lr); err != nil {
		return
	}

	db := models.GetDB()

	var user models.User
	result := db.First(&user, "name=?", lr.User)

	if result.Error != nil {
		c.IndentedJSON(http.StatusOK, gin.H{"status": "login failed", "message": result.Error})
		return
	}

	if user.CheckPassword(lr.Password) && user.Active {
		session := sessions.Default(c)
		session.Set("user", lr.User)
		session.Set("roles", strings.Join(user.Roles, ","))
		session.Save()
		c.IndentedJSON(http.StatusOK, gin.H{"message": "login success"})
	} else {
		c.IndentedJSON(http.StatusOK, gin.H{"message": "login failed"})
	}
}

// logout godoc
//
// @Summary	Logout
// @Desription	Logout user
// @Tags	user
// @Accept	json
// @Produce json
// @Success	200	json result
// @Router	/auth/logout	[get]
func (a AuthController) Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.IndentedJSON(http.StatusOK, gin.H{"status": "success"})
}

// status godoc
//
//	@Summary		Auth Status
//	@Description	Check login status
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Success		200		{string}	result
//	@Router			/auth/status [get]
func (a AuthController) Status(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("user")
	roles := session.Get("roles")

	if username == nil {
		c.IndentedJSON(http.StatusOK, gin.H{"message": "not logged in"})
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{"message": "logged in", "user": username, "roles": roles})
}

// users godoc
//
// @Summary List users
// @Description List users on the sytem
// @Tags	user
// @Accept json
// @Produce 	json
// @Success 200 json result
// @Router /auth/users [get]
func (a AuthController) ListUsers(c *gin.Context) {
	db := models.GetDB()
	var users []models.User
	db.Find(&users)

	c.IndentedJSON(http.StatusOK, users)

}

// delete user godoc
//
// @Summary delete user
// @Description delete a user
// @Tags user
// @Accept json
// @Produce json
// @Param uid path string true "User ID"
// @Success 200 {string} response
// @Router /auth/user/{uid} [delete]
func (a AuthController) DeleteUser(c *gin.Context) {
	db := models.GetDB()
	var user models.User
	result := db.First(&user, "UID=?", c.Param("uid"))
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.IndentedJSON(http.StatusOK, gin.H{"status": "error", "message": "user not found"})
			return
		} else {
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "db error"})
			return
		}
	} else {
		if user.Name != "admin" {
			result := db.Delete(&user)
			if result.Error != nil {
				c.IndentedJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": result.Error})
				return
			} else {
				c.IndentedJSON(http.StatusOK, gin.H{"status": "success", "message": "user deleted"})
				return
			}
		} else {
			c.IndentedJSON(http.StatusBadRequest, gin.H{"status": "error", "message": "refusing to delete admin user"})
			return
		}

	}
}

// update user godoc
//
// @Summary update user
// @Description update users attributes
// @Tags	user
// @Accept	json
// @Produce	json
// @Param	user	body		models.UserReq	true	"User Data"
// @Param	uid	path	string	true	"User ID"
// @Success 200 {string} response
// @Router /auth/users/{uid} [put]
func (a AuthController) UpdateUser(c *gin.Context) {
	db := models.GetDB()
	var user models.User
	result := db.First(&user, "UID=?", c.Param("uid"))
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.IndentedJSON(http.StatusOK, gin.H{"status": "error", "message": "user not found"})
			return
		} else {
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "db error"})
			return
		}
	} else {
		var userPost models.UserReq
		if err := c.BindJSON(&userPost); err != nil {
			return
		}
		user.Active = userPost.Active
		user.Roles = userPost.Roles
		result := db.Save(&user)
		if result.Error != nil {
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": result.Error})
			return
		} else {
			c.IndentedJSON(http.StatusOK, gin.H{"status": "success"})
		}
	}

}

// register godoc
//
// @Summary		Register User
// @Description	Register a user
// @Tags		user
// @Accept		json
// @Produces	json
// @Param		register	body		models.RegisterReq	true	"Login Data"
// @Success		200	{string} result
// @Router		/auth/register [post]
func (a AuthController) Register(c *gin.Context) {
	regreq := new(models.RegisterReq)
	db := models.GetDB()

	if err := c.BindJSON(&regreq); err != nil {
		return
	}
	if regreq.Name == "" || regreq.Password == "" {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"status": "error", "message": "bad response"})
		return
	}
	var user models.User
	result := db.First(&user, "name=?", regreq.Name)

	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			fmt.Println(("some other errer"))
			fmt.Println(result.Error)
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "db error"})
			return
		}
	}
	if result.RowsAffected != 0 {
		fmt.Println("user already exits")
		c.IndentedJSON(http.StatusOK, gin.H{"status": "error", "message": "user already exists"})
		return
	}

	fmt.Println("checked if user exists")
	newUser := models.MakeUser(regreq.Name)
	bytes, hasherr := bcrypt.GenerateFromPassword([]byte(regreq.Password), 14)
	if hasherr != nil {
		panic("error hashing password")
	}
	newUser.PasswordHash = string(bytes)
	newUser.Active = false
	result = db.Create(&newUser)
	if result.Error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"status": "error", "error": result.Error})
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{"status": "success", "message": "user created"})
}

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

	// Load HMAC secret from environment
	hmacSecret := os.Getenv("HMAC_SECRET")
	if hmacSecret == "" {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "server configuration error"})
		return
	}

	mac := hmac.New(sha256.New, []byte(hmacSecret))
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

	// Load HMAC secret from environment
	hmacSecret := os.Getenv("HMAC_SECRET")
	if hmacSecret == "" {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "server configuration error"})
		return
	}

	mac := hmac.New(sha256.New, []byte(hmacSecret))
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

// SignControl godoc
//
//	@Summary		Sign a control QR payload
//	@Schemes
//	@Description	Ed25519-sign a control command (e.g. setup-station) so cashier stations can verify it came from this server
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			signcontrol	body		models.SignControlReq	true	"control command data"
//	@Success		200			{object}	object
//	@Router			/auth/sign-control [post]
func (a AuthController) SignControl(c *gin.Context) {
	seedB64 := os.Getenv("CONTROL_SIGNING_KEY")
	if seedB64 == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "CONTROL_SIGNING_KEY not configured"})
		return
	}
	seed, err := base64.StdEncoding.DecodeString(seedB64)
	if err != nil || len(seed) != ed25519.SeedSize {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "CONTROL_SIGNING_KEY is malformed"})
		return
	}
	priv := ed25519.NewKeyFromSeed(seed)

	var req models.SignControlReq
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid request"})
		return
	}

	// Whitelist of signable control commands; extend deliberately.
	if req.Control != "setup-station" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "control command not signable"})
		return
	}
	if !stationRE.MatchString(req.Station) {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid station id"})
		return
	}

	ttl := 24 * time.Hour
	if v := os.Getenv("CONTROL_QR_TTL"); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil {
			ttl = parsed
		}
	}
	exp := time.Now().Add(ttl).Unix()

	payload, err := json.Marshal(map[string]interface{}{
		"control": req.Control,
		"station": req.Station,
		"exp":     exp,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to build payload"})
		return
	}

	sig := ed25519.Sign(priv, payload)
	envelope, err := json.Marshal(map[string]string{
		"c": base64.StdEncoding.EncodeToString(payload),
		"s": base64.StdEncoding.EncodeToString(sig),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to build envelope"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "qr": string(envelope), "payload": string(payload), "exp": exp})
}

// CreateAPIKey godoc
//
// @Summary Create API Key
// @Description Create a new API key for programmatic access
// @Tags auth
// @Accept json
// @Produce json
// @Param apikey body object true "API Key Data"
// @Success 200 {object} object
// @Router /auth/api-keys [post]
func (a AuthController) CreateAPIKey(c *gin.Context) {
	db := models.GetDB()

	var req struct {
		Name    string `json:"name" binding:"required"`
		Purpose string `json:"purpose" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid request"})
		return
	}

	// Validate purpose
	if req.Purpose != "scanner" && req.Purpose != "ordering" {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"status": "error", "message": "purpose must be 'scanner' or 'ordering'"})
		return
	}

	// Generate the API key
	plaintextKey, apiKey := models.GenerateAPIKey(req.Name, req.Purpose)

	// Save to database
	result := db.Create(&apiKey)
	if result.Error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to create API key"})
		return
	}

	// Return the plaintext key (only time it will be visible)
	c.IndentedJSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "API key created successfully. Save this key, it will not be shown again.",
		"api_key": plaintextKey,
		"key_id":  apiKey.KeyID,
		"name":    apiKey.Name,
		"purpose": apiKey.Purpose,
	})
}

// ListAPIKeys godoc
//
// @Summary List API Keys
// @Description List all API keys (keys are hashed, only metadata shown)
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {array} object
// @Router /auth/api-keys [get]
func (a AuthController) ListAPIKeys(c *gin.Context) {
	db := models.GetDB()

	var apiKeys []models.APIKey
	db.Find(&apiKeys)

	// Format response without exposing hashes
	var response []gin.H
	for _, key := range apiKeys {
		response = append(response, gin.H{
			"key_id":     key.KeyID,
			"name":       key.Name,
			"purpose":    key.Purpose,
			"active":     key.Active,
			"created_at": key.CreatedAt,
		})
	}

	c.IndentedJSON(http.StatusOK, response)
}

// RevokeAPIKey godoc
//
// @Summary Revoke API Key
// @Description Deactivate an API key
// @Tags auth
// @Accept json
// @Produce json
// @Param key_id path string true "Key ID"
// @Success 200 {object} object
// @Router /auth/api-keys/{key_id} [delete]
func (a AuthController) RevokeAPIKey(c *gin.Context) {
	db := models.GetDB()
	keyID := c.Param("key_id")

	var apiKey models.APIKey
	result := db.First(&apiKey, "key_id = ?", keyID)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.IndentedJSON(http.StatusNotFound, gin.H{"status": "error", "message": "API key not found"})
			return
		}
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "database error"})
		return
	}

	// Mark as inactive instead of deleting (audit trail)
	apiKey.Active = false
	result = db.Save(&apiKey)

	if result.Error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to revoke API key"})
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{"status": "success", "message": "API key revoked"})
}

// ValidateAPIKey godoc
//
// @Summary Validate API Key
// @Description Check if an API key is valid and active
// @Tags auth
// @Accept json
// @Produce json
// @Param apikey body object true "API Key to validate"
// @Success 200 {object} object
// @Router /auth/api-keys/validate [post]
func (a AuthController) ValidateAPIKey(c *gin.Context) {
	db := models.GetDB()

	var req struct {
		APIKey string `json:"api_key" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid request"})
		return
	}

	// Find all active API keys
	var apiKeys []models.APIKey
	if err := db.Where("active = ?", true).Find(&apiKeys).Error; err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "database error"})
		return
	}

	// Check each active key
	for _, apiKey := range apiKeys {
		if apiKey.ValidateKey(req.APIKey) {
			// Valid key found
			c.IndentedJSON(http.StatusOK, gin.H{
				"status":  "success",
				"valid":   true,
				"message": "API key is valid",
				"key_id":  apiKey.KeyID,
				"name":    apiKey.Name,
				"purpose": apiKey.Purpose,
			})
			return
		}
	}

	// No matching key found
	c.IndentedJSON(http.StatusOK, gin.H{
		"status":  "success",
		"valid":   false,
		"message": "API key is invalid or has been revoked",
	})
}

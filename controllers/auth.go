package controllers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/brian-l-johnson/CashierStatusBoard/v2/models"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthController struct{}

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

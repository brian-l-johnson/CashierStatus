package server

import (
	"html/template"
	"net/http"
	"os"
	"strings"

	"github.com/brian-l-johnson/CashierStatusBoard/v2/controllers"
	docs "github.com/brian-l-johnson/CashierStatusBoard/v2/docs"
	"github.com/brian-l-johnson/CashierStatusBoard/v2/middleware"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func getAPIBaseURL() string {
	return os.Getenv("API_BASE_URL")
}

func isAdmin(roles string) bool {
	return strings.Contains(roles, "admin")
}

func NewRouter() *gin.Engine {
	router := gin.New()

	store := cookie.NewStore([]byte(os.Getenv("SESSION_SECRET")))
	router.Use(sessions.Sessions("session", store))

	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	health := new(controllers.HealthController)
	router.GET("/health", health.Status)

	cashiers := new(controllers.CashierController)
	router.GET("/cashiers", cashiers.GetCashiers)
	router.POST("/cashiers", middleware.Authorize("admin"), cashiers.CreateCashier)
	router.PATCH("/cashiers/:cid", middleware.Authorize("update"), cashiers.UpdateCashier)
	router.DELETE("/cashiers/:cid", middleware.Authorize("admin"), cashiers.DeleteCashier)
	router.GET("/cashiers/getupdate-ws", cashiers.GetCashierUpdates)

	auth := new(controllers.AuthController)
	router.POST("/auth/mac", middleware.Authorize("update"), auth.Mac)
	router.POST("/auth/verify", middleware.Authorize("view"), auth.Verify)
	router.POST("/auth/login", auth.Login)
	router.GET("/auth/logout", auth.Logout)
	router.GET("/auth/status", auth.Status)
	router.POST("/auth/register", auth.Register)
	router.GET("/auth/users", middleware.Authorize("admin"), auth.ListUsers)
	router.PUT("/auth/users/:uid", middleware.Authorize("admin"), auth.UpdateUser)
	router.DELETE("/auth/user/:uid", middleware.Authorize("admin"), auth.DeleteUser)

	docs.SwaggerInfo.BasePath = "/"

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	//static
	router.Static("/static", "./static")

	router.SetFuncMap(template.FuncMap{
		"getAPIBaseURL": getAPIBaseURL,
		"isAdmin":       isAdmin,
	})

	//templates
	router.LoadHTMLGlob("templates/*")

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "main.html", gin.H{"title": "Now Serving"})
	})
	router.GET("/status", func(c *gin.Context) {
		c.HTML(http.StatusOK, "status.html", gin.H{})
	})
	router.GET("/update", middleware.Authorize("update"), func(c *gin.Context) {
		session := sessions.Default(c)
		user := session.Get("user")
		roles := session.Get("roles")

		c.HTML(http.StatusOK, "manual.html", gin.H{"title": "Manual Update", "user": user, "roles": roles})
	})
	router.GET("/qrtester", func(c *gin.Context) {
		c.HTML(http.StatusOK, "qrtester.html", gin.H{"title": "QRTester"})
	})
	router.GET("/ordertester", func(c *gin.Context) {
		c.HTML(http.StatusOK, "ordertester.html", gin.H{"title": "QRTester"})
	})
	router.GET("/cashiersetup", middleware.Authorize("admin"), func(c *gin.Context) {
		session := sessions.Default(c)
		user := session.Get("user")
		roles := session.Get("roles")
		c.HTML(http.StatusOK, "cashiersetup.html", gin.H{"title": "Cashier Setup", "user": user, "roles": roles})
	})
	router.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", gin.H{"title": "Login"})
	})
	router.GET("/register", func(c *gin.Context) {
		c.HTML(http.StatusOK, "register.html", gin.H{"title": "register"})
	})
	router.GET("/users", middleware.Authorize("admin"), func(c *gin.Context) {
		session := sessions.Default(c)
		user := session.Get("user")
		roles := session.Get("roles")
		c.HTML(http.StatusOK, "users.html", gin.H{"title": "register", "user": user, "roles": roles})
	})
	router.GET("/logout.html", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Clear()
		session.Save()
		c.Redirect(http.StatusFound, "/login")
	})

	return router
}

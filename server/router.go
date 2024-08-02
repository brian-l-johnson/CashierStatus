package server

import (
	"html/template"
	"net/http"
	"os"

	"github.com/brian-l-johnson/CashierStatusBoard/v2/controllers"
	docs "github.com/brian-l-johnson/CashierStatusBoard/v2/docs"
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func getAPIBaseURL() string {
	return os.Getenv("API_BASE_URL")
}

func NewRouter() *gin.Engine {
	router := gin.New()

	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	health := new(controllers.HealthController)
	router.GET("/health", health.Status)

	cashiers := new(controllers.CashierController)
	router.GET("/cashiers", cashiers.GetCashiers)
	router.POST("/cashiers", cashiers.CreateCashier)
	router.PATCH("/cashiers/:cid", cashiers.UpdateCashier)
	router.DELETE("/cashiers/:cid", cashiers.DeleteCashier)
	router.GET("/cashiers/getupdate-ws", cashiers.GetCashierUpdates)

	auth := new(controllers.AuthController)
	router.POST("/auth/mac", auth.Mac)
	router.POST("/auth/verify", auth.Verify)

	docs.SwaggerInfo.BasePath = "/"

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	//static
	router.Static("/static", "./static")

	router.SetFuncMap(template.FuncMap{
		"getAPIBaseURL": getAPIBaseURL,
	})

	//templates
	router.LoadHTMLGlob("templates/*")

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "main.html", gin.H{"title": "Now Serving"})
	})
	router.GET("/update", func(c *gin.Context) {
		c.HTML(http.StatusOK, "manual.html", gin.H{"title": "Manual Update"})
	})
	router.GET("/qrtester", func(c *gin.Context) {
		c.HTML(http.StatusOK, "qrtester.html", gin.H{"title": "QRTester"})
	})
	router.GET("/ordertester", func(c *gin.Context) {
		c.HTML(http.StatusOK, "ordertester.html", gin.H{"title": "QRTester"})
	})
	router.GET("/cashiersetup", func(c *gin.Context) {
		c.HTML(http.StatusOK, "cashiersetup.html", gin.H{"title": "Cashier Setup"})
	})

	return router
}

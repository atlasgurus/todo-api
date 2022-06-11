package app

import (
	"github.com/gin-gonic/gin"
	_ "github.com/go-redis/redis/v7"
	_ "github.com/lib/pq"
	"github.com/tintash-training/todo-api/app/authentication"
	"github.com/tintash-training/todo-api/app/config"
	"github.com/tintash-training/todo-api/app/models"
	_ "github.com/twinj/uuid"
	"log"
	"net/http"
)

type App struct {
	router *gin.Engine
	auth   *authentication.Auth
	config *config.Config
	//db     models.Datastore
}

func (app *App) Start(config *config.Config) {
	auth, err := authentication.CreateAuthenticator(config.RedisConfig)
	if err != nil {
		panic(err)
	}

	app.config = config
	app.router = gin.Default()
	app.auth = auth
	app.initRouters()

	app.run(":8080")
}

func (app *App) run(addr string) {
	log.Fatal(app.router.Run(addr))
}

func (app *App) initRouters() {
	app.router.POST("/login", app.Login)
	app.router.POST("/migrate", app.CreateTables)
	app.router.POST("/todo", TokenAuthMiddleware(), app.CreateTodo)
	app.router.POST("/logout", TokenAuthMiddleware(), app.Logout)
}

func (app *App) Login(c *gin.Context) {
	var u models.User
	if err := c.ShouldBindJSON(&u); err != nil {
		c.JSON(http.StatusUnprocessableEntity, "Invalid json provided")
		return
	}

	db, err := models.ConnectDS(app.config.DBConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "invalid json")
		return
	}
	user, err := db.ReadUser(u.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, "Please provide valid login details")
		return
	}

	//compare the user from the request, with the one we defined:
	if user.Username != u.Username || user.Password != u.Password {
		c.JSON(http.StatusUnauthorized, "Please provide valid login details")
		return
	}

	ts, err := authentication.CreateToken(user.ID)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, err.Error())
		return
	}
	saveErr := app.auth.CreateAuth(user.ID, ts)
	if saveErr != nil {
		c.JSON(http.StatusUnprocessableEntity, saveErr.Error())
	}
	tokens := map[string]string{
		"access_token":  ts.AccessToken,
		"refresh_token": ts.RefreshToken,
	}
	c.JSON(http.StatusOK, tokens)
}

func (app *App) CreateTodo(c *gin.Context) {
	var td *models.Todo
	if err := c.ShouldBindJSON(&td); err != nil {
		c.JSON(http.StatusUnprocessableEntity, "invalid json")
		return
	}
	tokenAuth, err := authentication.ExtractTokenMetadata(c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, "unauthorized")
		return
	}
	userId, err := app.auth.FetchAuth(tokenAuth)
	if err != nil {
		c.JSON(http.StatusUnauthorized, "unauthorized")
		return
	}
	td.UserID = userId

	db, err := models.ConnectDS(app.config.DBConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "unable to connect to the DB")
		return
	}

	err = db.SaveToDo(td)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "unable to save todo to the DB")
		return
	}

	//you can proceed to save the Todo to a database
	//but we will just return it to the caller here:
	c.JSON(http.StatusCreated, td)
}

func (app *App) Logout(c *gin.Context) {
	au, err := authentication.ExtractTokenMetadata(c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, "unauthorized")
		return
	}
	deleted, delErr := app.auth.DeleteAuth(au.AccessUuid)
	if delErr != nil || deleted == 0 { //if any goes wrong
		c.JSON(http.StatusUnauthorized, "unauthorized")
		return
	}
	c.JSON(http.StatusOK, "Successfully logged out")
}

func (app *App) CreateTables(c *gin.Context) {
	ds, err := models.ConnectDS(app.config.DBConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "unable to connect to the DB")
		return
	}

	err = ds.CreateTables()
	if err != nil {
		c.JSON(http.StatusInternalServerError, "unable to create tables")
		return
	}
	c.JSON(http.StatusOK, "Successfully initialized DB tables")
}

func TokenAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		err := authentication.TokenValid(c.Request)
		if err != nil {
			c.JSON(http.StatusUnauthorized, err.Error())
			c.Abort()
			return
		}
		c.Next()
	}
}

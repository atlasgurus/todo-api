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
	"strconv"
)

type App struct {
	router *gin.Engine
	auth   *authentication.Auth
	config *config.Config
	//db     models.Datastore
}

func (app *App) Start(config *config.Config) {
	auth, err := authentication.CreateAuthenticator(config.AuthConfig)
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
	app.router.POST("/register", app.Register)
	app.router.POST("/login", app.Login)
	app.router.POST("/migrate", app.CreateTables)
	app.router.POST("/add-task", TokenAuthMiddleware(), app.CreateTodo)
	app.router.PUT("/update-task/:task-id", TokenAuthMiddleware(), app.UpdateTodo)
	app.router.DELETE("/delete-task/:task-id", TokenAuthMiddleware(), app.DeleteTodo)
	app.router.GET("/list-tasks", TokenAuthMiddleware(), app.GetAllTasks)
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
		c.Status(http.StatusInternalServerError)
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

	ts, err := app.auth.CreateToken(user.ID)
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

func (app *App) UpdateTodo(c *gin.Context) {
	var ntd *models.NewTodo
	if err := c.ShouldBindJSON(&ntd); err != nil {
		c.JSON(http.StatusUnprocessableEntity, "invalid json")
		return
	}
	taskIdStr := c.Param("task-id")
	taskId, err := strconv.ParseUint(taskIdStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, "no valid task-id")
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

	db, err := models.ConnectDS(app.config.DBConfig)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	td := models.Todo{ID: taskId, NewTodo: *ntd, UserID: userId}

	rows, err := db.UpdateToDo(&td)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	switch rows {
	case 0:
		c.JSON(http.StatusNotFound, "task not found")
	case 1:
		c.Status(http.StatusCreated)
	default:
		panic("should not happen")
	}
}

func (app *App) CreateTodo(c *gin.Context) {
	var ntd *models.NewTodo
	if err := c.ShouldBindJSON(&ntd); err != nil {
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

	td := models.Todo{NewTodo: *ntd, UserID: userId}

	db, err := models.ConnectDS(app.config.DBConfig)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	err = db.SaveToDo(&td)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	response := gin.H{"task-id": td.ID}

	c.JSON(http.StatusCreated, response)
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
		c.Status(http.StatusInternalServerError)
		return
	}

	err = ds.CreateTables()
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, "Successfully initialized DB tables")
}

func (app *App) Register(c *gin.Context) {
	var u models.NewUser
	if err := c.ShouldBindJSON(&u); err != nil {
		c.JSON(http.StatusUnprocessableEntity, "Invalid json provided")
		return
	}

	db, err := models.ConnectDS(app.config.DBConfig)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	err = db.CreateUser(&u)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, "User created successfully")
}

func (app *App) GetAllTasks(c *gin.Context) {
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

	db, err := models.ConnectDS(app.config.DBConfig)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	tasks, err := db.GetAllTasks(userId)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, tasks)
}

func (app *App) DeleteTodo(c *gin.Context) {
	taskIdStr := c.Param("task-id")
	taskId, err := strconv.ParseUint(taskIdStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, "no valid task-id")
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

	db, err := models.ConnectDS(app.config.DBConfig)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	rows, err := db.DeleteToDo(userId, taskId)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	switch rows {
	case 0:
		c.JSON(http.StatusNotFound, "task not found")
	case 1:
		c.Status(http.StatusCreated)
	default:
		panic("should not happen")
	}
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

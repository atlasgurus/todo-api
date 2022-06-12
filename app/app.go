package app

import (
	"crypto/tls"
	"github.com/gin-gonic/gin"
	"github.com/go-gomail/gomail"
	_ "github.com/go-redis/redis/v7"
	_ "github.com/lib/pq"
	"github.com/tintash-training/todo-api/app/authentication"
	"github.com/tintash-training/todo-api/app/config"
	"github.com/tintash-training/todo-api/app/models"
	_ "github.com/twinj/uuid"
	"log"
	"net/http"
	"strconv"
	"strings"
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
	app.router.POST("/assign-task", TokenAuthMiddleware(), app.AssignTodo)
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
	user, err := db.ReadUser(u.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, "Please provide valid login details")
		return
	}

	//compare the user from the request, with the one we defined:
	if strings.ToLower(u.Email) != user.Email || u.Password != user.Password {
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

func (app *App) AssignTodo(c *gin.Context) {
	var atd *models.AssignedTodo
	if err := c.ShouldBindJSON(&atd); err != nil {
		c.JSON(http.StatusUnprocessableEntity, "invalid json")
		return
	}
	tokenAuth, err := authentication.ExtractTokenMetadata(c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, "unauthorized")
		return
	}
	_, err = app.auth.FetchAuth(tokenAuth)
	if err != nil {
		c.JSON(http.StatusUnauthorized, "unauthorized")
		return
	}

	db, err := models.ConnectDS(app.config.DBConfig)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	// Lookup the user by email
	user, err := db.ReadUser(atd.Email)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	if user == nil {
		// User not registered.  Create a temporary registration and notify the user by email.
		Pending := true
		newUser := &models.NewUser{Email: atd.Email, Pending: &Pending}

		err = db.CreateUser(newUser)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		user, err = db.ReadUser(atd.Email)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		err = app.sendRegistrationEmail(atd)
		if err != nil {
			// TODO this failed request will have a side effect of having created a user.  Roll it back?
			c.Status(http.StatusInternalServerError)
			return
		}
	}

	td := models.Todo{NewTodo: atd.NewTodo, UserID: user.ID}

	err = db.SaveToDo(&td)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	response := gin.H{"task-id": td.ID}

	c.JSON(http.StatusCreated, response)
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
	user, err := db.ReadUser(u.Email)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	if user == nil {
		// This is a brand new user
		err = db.CreateUser(&u)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.JSON(http.StatusOK, "User created successfully")
	} else if *user.Pending {
		err = db.UpdateUser(&u)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.JSON(http.StatusOK, "User created successfully")
	} else {
		c.JSON(http.StatusConflict, "User already exists")
	}

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

func (app *App) sendRegistrationEmail(atd *models.AssignedTodo) error {
	m := gomail.NewMessage()

	// Set E-Mail sender
	m.SetHeader("From", "from@gmail.com")

	// Set E-Mail receivers
	m.SetHeader("To", "to@example.com")

	// Set E-Mail subject
	m.SetHeader("Subject", "A task has been assigned to you")

	// Set E-Mail body. You can set plain text or html with text/html
	m.SetBody("text/plain", atd.Title)

	// Settings for SMTP server
	d := gomail.NewDialer("smtp.gmail.com", 587, "from@gmail.com", "<email_password>")

	// This is only needed when SSL/TLS certificate is not valid on server.
	// In production this should be set to false.
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	// Now send E-Mail
	err := d.DialAndSend(m)
	return err
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

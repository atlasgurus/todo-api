package models

import (
	"database/sql"
	"fmt"
	"github.com/tintash-training/todo-api/app/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Datastore interface {
	//AllTodos() ([]*Todo, error)
	//AddTodo(string, string) (*Todo, error)
	//GetTodo(int) (*Todo, error)

	SaveToDo(td *Todo) error
	UpdateToDo(td *Todo) (int64, error)
	DeleteToDo(user uint64, taskId uint64) (int64, error)
	GetAllTasks(userId uint64) ([]Todo, error)
	ReadUser(username string) (user *User, err error)
	CreateTables() error
	CreateUser(user *NewUser) error
}

type GormDB struct {
	*gorm.DB
}

type SqlDB struct {
	*sql.DB
}

func (db *GormDB) CreateTables() error {
	users := []User{
		{NewUser: NewUser{Username: "Paul", Password: "password"}},
		{NewUser: NewUser{Username: "Paul", Password: "password"}}}
	result := db.Create(&users) // pass pointer of data to Create
	return result.Error
}

// ReadUser database/sql implementation
func (db *SqlDB) ReadUser(username string) (user *User, err error) {
	user = &User{}
	query := fmt.Sprintf("SELECT * FROM users WHERE username = '%s'", username)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}

	if rows.Next() {
		err = rows.Scan(&user.ID, &user.Username, &user.Password)
	}
	return
}

// ReadUser gorm implementation
func (db *GormDB) ReadUser(username string) (user *User, err error) {
	user = &User{}
	result := db.First(user, "username = ?", username)
	err = result.Error
	return
}

func makeDataSourceName(config *config.DBConfig) string {
	return fmt.Sprintf("host=%s user=%s dbname=%s sslmode=%s port=%s password=%s",
		config.Host,
		config.Username,
		config.Name,
		config.SSLMode,
		config.Port,
		config.Password)
}

func connectGormDB(config *config.DBConfig) (ds Datastore, err error) {
	var db *gorm.DB
	db, err = gorm.Open(postgres.Open(makeDataSourceName(config)), &gorm.Config{})
	if err != nil {
		return
	}
	err = db.AutoMigrate(&User{}, &Todo{})
	if err != nil {
		return
	}

	ds = Datastore(&GormDB{db})
	return
}

func connectSqlDB(config *config.DBConfig) (ds Datastore, err error) {
	var db *sql.DB
	db, err = sql.Open("postgres", makeDataSourceName(config))
	ds = Datastore(&SqlDB{db})
	return
}

func ConnectDS(config *config.DBConfig) (ds Datastore, err error) {
	switch config.Impl {
	case "gorm":
		return connectGormDB(config)
	case "sql":
		return connectSqlDB(config)
	default:
		panic(config.Impl)
	}
}

func (db *SqlDB) CreateTables() error {
	query := `
		DROP TABLE IF EXISTS todos;
		DROP TABLE IF EXISTS users;
		CREATE TABLE users (
		    id SERIAL PRIMARY KEY,
   			username varchar,
   			password varchar
		); `
	_, err := db.Query(query)
	if err != nil {
		return err
	}

	query = `
		INSERT INTO users (username, password) VALUES ( 'Paul', 'password');
		INSERT INTO users (username, password) VALUES ( 'John', 'password'); `

	_, err = db.Query(query)
	if err != nil {
		return err
	}

	query = `
		CREATE TABLE todos (
   		userid INT,
   		title varchar,
   		CONSTRAINT fk_user
      		FOREIGN KEY(userid) 
	  		REFERENCES users(id)
		); `

	_, err = db.Query(query)
	if err != nil {
		return err
	}

	return err
}

func (db *GormDB) SaveToDo(td *Todo) error {
	result := db.Create(td) // pass pointer of data to Create
	return result.Error
}

func (db *GormDB) UpdateToDo(td *Todo) (int64, error) {
	result := db.Model(&Todo{}).Where("ID = ? and userid = ?", td.ID, td.UserID).Updates(
		Todo{NewTodo: NewTodo{Title: td.Title}})

	return result.RowsAffected, result.Error
}

func (db *GormDB) GetAllTasks(userId uint64) ([]Todo, error) {
	todos := []Todo{}
	result := db.Where("userid = ?", userId).Find(&todos)
	return todos, result.Error
}

func (db *SqlDB) GetAllTasks(userId uint64) ([]Todo, error) {
	return nil, fmt.Errorf("not implemented")
}

func (db *GormDB) DeleteToDo(userId uint64, taskId uint64) (int64, error) {
	result := db.Where("ID = ? and userid = ?", taskId, userId).Delete(&Todo{})

	return result.RowsAffected, result.Error
}

func (db *SqlDB) DeleteToDo(userId uint64, taskId uint64) (int64, error) {
	return 0, fmt.Errorf("not implemented")
}

func (db *SqlDB) SaveToDo(td *Todo) error {
	_, err := db.Query("INSERT INTO todos (userid, title) VALUES ($1, $2);", td.UserID, td.Title)
	return err
}

func (db *SqlDB) UpdateToDo(td *Todo) (int64, error) {
	return 0, fmt.Errorf("not implemented")
}

func (db *SqlDB) CreateUser(user *NewUser) error {
	return fmt.Errorf("not implemented")
}

func (db *GormDB) CreateUser(user *NewUser) error {
	u := User{NewUser: *user}
	result := db.Create(&u)
	return result.Error
}

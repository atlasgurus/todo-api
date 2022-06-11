package models

import (
	"gorm.io/gorm"
	"time"
)

type NewUser struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	FirstName string `json:"first-name"`
	LastName  string `json:"last-name"`
	Password  string `json:"password"`
}

type User struct {
	ID uint64 `json:"id"`
	NewUser
}

type NewTodo struct {
	ID    uint64 `json:"-"`
	Title string `json:"title"`
}

type Todo struct {
	ID        uint64         `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"-"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	NewTodo
	UserID uint64 `gorm:"column:userid" gorm:"index" json:"userid"`
}

package models

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Email      string
	TelegramID int64 `gorm:"uniqueIndex"`
	FirstName  string
	LastName   string
	Username   string
	PhotoURL   string
}

type Token struct {
	gorm.Model
	TokenString string `gorm:"uniqueIndex"` // Deprecated: for backward compatibility
	TokenHash   string `gorm:"uniqueIndex"` // SHA256 hash of the token
	UserID      uint
	User        User
}

type Domain struct {
	gorm.Model
	Name   string `gorm:"uniqueIndex"`
	UserID uint
	User   User
}

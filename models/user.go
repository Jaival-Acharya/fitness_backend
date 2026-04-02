package models

import "time"

// User represents a system user account
type User struct {
	ID        int        `json:"id" gorm:"primaryKey"`
	Email     string     `json:"email" gorm:"uniqueIndex"`
	Password  string     `json:"-" gorm:"column:password_hash"`
	FirstName string     `json:"first_name"`
	LastName  string     `json:"last_name"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at" gorm:"index"`
}

func (User) TableName() string {
	return "users"
}

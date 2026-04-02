package models

import "time"

// UserProfile stores health metrics for a user
type UserProfile struct {
	ID        int       `json:"id" gorm:"primaryKey"`
	UserID    int       `json:"user_id" gorm:"uniqueIndex"`
	Height    float64   `json:"height"` // in cm
	Weight    float64   `json:"weight"` // in kg
	Age       int       `json:"age"`
	Gender    string    `json:"gender"` // 'Male', 'Female', 'Other'
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (UserProfile) TableName() string {
	return "user_profiles"
}

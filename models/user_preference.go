package models

import "time"

// UserPreference stores fitness goals and dietary preferences
type UserPreference struct {
	ID           int       `json:"id" gorm:"primaryKey"`
	UserID       int       `json:"user_id" gorm:"uniqueIndex"`
	DietType     string    `json:"diet_type"`            // 'veg', 'egg', 'nonveg'
	Goal         string    `json:"fitness_goal"`         // 'lose', 'maintain', 'build'
	Activity     string    `json:"activity_level"`       // 'sedentary', 'light', 'moderate', 'very_active'
	Restrictions string    `json:"dietary_restrictions"` // allergies, restrictions
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (UserPreference) TableName() string {
	return "user_preferences"
}

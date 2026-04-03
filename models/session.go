package models

import (
	"database/sql/driver"
	"time"

	"gorm.io/datatypes"
)

type Session struct {
	ID                 string         `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	BMI                float64        `json:"bmi"`
	Weight             float64        `json:"weight"`
	Height             float64        `json:"height"`
	Age                int            `json:"age"`
	Gender             string         `json:"gender"`
	ActivityLevel      string         `json:"activity_level"`
	FitnessGoal        string         `json:"fitness_goal"`
	DietType           string         `json:"diet_type"`
	HeartRate          int            `json:"heart_rate"`
	Allergies          datatypes.JSON `json:"allergies" gorm:"type:jsonb;default:'[]'"`
	RestrictionsText   string         `json:"restrictions_text" gorm:"column:restrictions_text"`
	TargetWeight       *float64       `json:"target_weight"`
	DailyCalorieTarget int            `json:"daily_calorie_target" gorm:"default:2200"`
	DailyProteinTarget float64        `json:"daily_protein_target" gorm:"default:150"`
	DailyCarbsTarget   float64        `json:"daily_carbs_target" gorm:"default:275"`
	DailyFatTarget     float64        `json:"daily_fat_target" gorm:"default:73"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

func (Session) TableName() string {
	return "sessions"
}

func (s Session) Value() (driver.Value, error) {
	return s, nil
}

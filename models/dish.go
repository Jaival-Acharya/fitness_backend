package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/datatypes"
)

type Dish struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	Name         string         `json:"name"`
	DietType     string         `json:"diet_type"` // veg, egg, nonveg
	MealType     string         `json:"meal_type"` // Breakfast, Lunch, Dinner, Snack
	Description  string         `json:"description"`
	Emoji        string         `json:"emoji"`
	Calories     int            `json:"calories"`
	Protein      float64        `json:"protein"`
	Carbs        float64        `json:"carbs"`
	Fat          float64        `json:"fat"`
	PrepTime     string         `json:"prep_time"`
	Difficulty   string         `json:"difficulty"` // Easy, Medium, Hard
	Ingredients  datatypes.JSON `json:"ingredients" gorm:"type:jsonb"`
	Steps        datatypes.JSON `json:"steps" gorm:"type:jsonb"`
	AllergenTags datatypes.JSON `json:"allergen_tags" gorm:"type:jsonb;default:'[]'"`
	ImageURL     string         `json:"image_url"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

func (Dish) TableName() string {
	return "dishes"
}

func (d Dish) Value() (driver.Value, error) {
	return json.Marshal(d)
}

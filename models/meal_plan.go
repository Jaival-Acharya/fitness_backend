package models

import (
	"time"
)

type MealPlan struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	SessionID       string    `json:"session_id" gorm:"type:uuid;index"`
	MealDate        time.Time `json:"meal_date" gorm:"index"`
	BreakfastDishID *uint     `json:"breakfast_dish_id"`
	LunchDishID     *uint     `json:"lunch_dish_id"`
	DinnerDishID    *uint     `json:"dinner_dish_id"`
	TotalCalories   int       `json:"total_calories"`
	TotalProtein    float64   `json:"total_protein"`
	TotalCarbs      float64   `json:"total_carbs"`
	TotalFat        float64   `json:"total_fat"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Relationships (optional)
	BreakfastDish *Dish `json:"breakfast_dish,omitempty" gorm:"foreignKey:BreakfastDishID"`
	LunchDish     *Dish `json:"lunch_dish,omitempty" gorm:"foreignKey:LunchDishID"`
	DinnerDish    *Dish `json:"dinner_dish,omitempty" gorm:"foreignKey:DinnerDishID"`
}

func (MealPlan) TableName() string {
	return "meal_plans"
}

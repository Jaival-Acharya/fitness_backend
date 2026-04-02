package models

import "time"

// UserMeal tracks meals consumed by a user
type UserMeal struct {
	ID        int       `json:"id" gorm:"primaryKey"`
	UserID    int       `json:"user_id" gorm:"index"`
	RecipeID  int       `json:"recipe_id"`
	MealDate  string    `json:"meal_date"` // date in YYYY-MM-DD format
	CreatedAt time.Time `json:"created_at"`

	// Relation
	Recipe *Recipe `json:"recipe" gorm:"foreignKey:RecipeID"`
}

func (UserMeal) TableName() string {
	return "user_meals"
}

// UserDailyTarget stores daily calorie targets
type UserDailyTarget struct {
	ID            int       `json:"id" gorm:"primaryKey"`
	UserID        int       `json:"user_id" gorm:"index"`
	TargetDate    string    `json:"target_date"` // date in YYYY-MM-DD format
	CalorieTarget int       `json:"calorie_target"`
	CreatedAt     time.Time `json:"created_at"`
}

func (UserDailyTarget) TableName() string {
	return "user_daily_targets"
}

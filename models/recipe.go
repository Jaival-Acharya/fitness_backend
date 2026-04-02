package models

import "time"

// Recipe represents a dish/meal in the database
type Recipe struct {
	ID          int       `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name"`
	Emoji       string    `json:"emoji"`
	MealType    string    `json:"meal_type"` // 'breakfast', 'lunch', 'dinner', 'snack'
	Calories    int       `json:"calories"`
	PrepTime    string    `json:"prep_time"`  // e.g., '15 min'
	Difficulty  string    `json:"difficulty"` // 'easy', 'medium', 'hard'
	Description string    `json:"description"`
	DietType    string    `json:"diet_type"` // 'veg', 'egg', 'nonveg'
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relations
	Macros      *RecipeMacro       `json:"macros" gorm:"foreignKey:RecipeID"`
	Ingredients []RecipeIngredient `json:"ingredients" gorm:"foreignKey:RecipeID"`
	Steps       []RecipeStep       `json:"steps" gorm:"foreignKey:RecipeID"`
}

func (Recipe) TableName() string {
	return "recipes"
}

// RecipeMacro stores nutritional information per recipe
type RecipeMacro struct {
	ID        int       `json:"id" gorm:"primaryKey"`
	RecipeID  int       `json:"recipe_id" gorm:"uniqueIndex"`
	Protein   float64   `json:"protein"` // grams
	Carbs     float64   `json:"carbs"`   // grams
	Fat       float64   `json:"fat"`     // grams
	Fiber     float64   `json:"fiber"`   // grams
	CreatedAt time.Time `json:"created_at"`
}

func (RecipeMacro) TableName() string {
	return "recipe_macros"
}

// RecipeIngredient represents an ingredient in a recipe
type RecipeIngredient struct {
	ID             int       `json:"id" gorm:"primaryKey"`
	RecipeID       int       `json:"recipe_id"`
	IngredientName string    `json:"ingredient_name"`
	Quantity       float64   `json:"quantity"`
	Unit           string    `json:"unit"` // 'grams', 'ml', 'cups', etc.
	CreatedAt      time.Time `json:"created_at"`
}

func (RecipeIngredient) TableName() string {
	return "recipe_ingredients"
}

// RecipeStep represents a cooking step
type RecipeStep struct {
	ID          int       `json:"id" gorm:"primaryKey"`
	RecipeID    int       `json:"recipe_id"`
	StepNumber  int       `json:"step_number"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

func (RecipeStep) TableName() string {
	return "recipe_steps"
}

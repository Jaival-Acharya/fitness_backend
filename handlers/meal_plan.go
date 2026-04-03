package handlers

import (
	"fitfuel/db"
	"fitfuel/models"
	"fitfuel/services"
	"fmt"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
)

type MealPlanSuggestRequest struct {
	SessionID   string `json:"session_id"`
	FitnessGoal string `json:"fitness_goal"`
	DietType    string `json:"diet_type"`
}

type MealPlanSaveRequest struct {
	SessionID       string `json:"session_id"`
	BreakfastDishID *uint  `json:"breakfast_dish_id"`
	LunchDishID     *uint  `json:"lunch_dish_id"`
	DinnerDishID    *uint  `json:"dinner_dish_id"`
}

type MealPlanResponse struct {
	ID            uint         `json:"id"`
	SessionID     string       `json:"session_id"`
	BreakfastDish *models.Dish `json:"breakfast_dish"`
	LunchDish     *models.Dish `json:"lunch_dish"`
	DinnerDish    *models.Dish `json:"dinner_dish"`
	TotalCalories int          `json:"total_calories"`
	TotalProtein  float64      `json:"total_protein"`
	TotalCarbs    float64      `json:"total_carbs"`
	TotalFat      float64      `json:"total_fat"`
}

// SuggestMealPlan generates meal plan suggestions based on session fitness goal and diet type
func SuggestMealPlan(c *fiber.Ctx) error {
	var req MealPlanSuggestRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.SessionID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Session ID is required",
		})
	}

	// Get session to retrieve calorie/macro targets
	var session models.Session
	if err := db.DB.First(&session, "id = ?", req.SessionID).Error; err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Session not found",
		})
	}

	// Get breakfast, lunch, dinner dishes matching the diet type and calorie/macro ranges
	breakfastDish, err := getDishForMeal("Breakfast", req.DietType, session.DailyCalorieTarget)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to find breakfast dish: %v", err),
		})
	}

	lunchDish, err := getDishForMeal("Lunch", req.DietType, session.DailyCalorieTarget)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to find lunch dish: %v", err),
		})
	}

	dinnerDish, err := getDishForMeal("Dinner", req.DietType, session.DailyCalorieTarget)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to find dinner dish: %v", err),
		})
	}

	// Calculate totals
	totalCalories := breakfastDish.Calories + lunchDish.Calories + dinnerDish.Calories
	totalProtein := breakfastDish.Protein + lunchDish.Protein + dinnerDish.Protein
	totalCarbs := breakfastDish.Carbs + lunchDish.Carbs + dinnerDish.Carbs
	totalFat := breakfastDish.Fat + lunchDish.Fat + dinnerDish.Fat

	return c.Status(http.StatusOK).JSON(MealPlanResponse{
		SessionID:     req.SessionID,
		BreakfastDish: breakfastDish,
		LunchDish:     lunchDish,
		DinnerDish:    dinnerDish,
		TotalCalories: totalCalories,
		TotalProtein:  totalProtein,
		TotalCarbs:    totalCarbs,
		TotalFat:      totalFat,
	})
}

// SuggestWeekMealPlan generates meal plan suggestions for 1-7 days without repetition
func SuggestWeekMealPlan(c *fiber.Ctx) error {
	var req SuggestWeekMealPlanRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.SessionID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Session ID is required",
		})
	}

	if req.Days < 1 || req.Days > 7 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Days must be between 1 and 7",
		})
	}

	// Get session to retrieve calorie/macro targets
	var session models.Session
	if err := db.DB.First(&session, "id = ?", req.SessionID).Error; err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Session not found",
		})
	}

	weekPlans := []DayMealPlan{}
	usedDishIDs := make(map[uint]bool) // Track used dishes to avoid repetition
	weekCalories := 0
	weekProtein := 0.0
	weekCarbs := 0.0
	weekFat := 0.0

	// Generate meal plans for each day
	for day := 1; day <= req.Days; day++ {
		breakfastDish, err := getDishForMealWithExclusion("Breakfast", req.DietType, session.DailyCalorieTarget, usedDishIDs)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to find breakfast dish for day %d: %v", day, err),
			})
		}

		lunchDish, err := getDishForMealWithExclusion("Lunch", req.DietType, session.DailyCalorieTarget, usedDishIDs)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to find lunch dish for day %d: %v", day, err),
			})
		}

		dinnerDish, err := getDishForMealWithExclusion("Dinner", req.DietType, session.DailyCalorieTarget, usedDishIDs)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to find dinner dish for day %d: %v", day, err),
			})
		}

		// Mark dishes as used
		usedDishIDs[breakfastDish.ID] = true
		usedDishIDs[lunchDish.ID] = true
		usedDishIDs[dinnerDish.ID] = true

		// Calculate totals for this day
		dayCalories := breakfastDish.Calories + lunchDish.Calories + dinnerDish.Calories
		dayProtein := breakfastDish.Protein + lunchDish.Protein + dinnerDish.Protein
		dayCarbs := breakfastDish.Carbs + lunchDish.Carbs + dinnerDish.Carbs
		dayFat := breakfastDish.Fat + lunchDish.Fat + dinnerDish.Fat

		weekPlans = append(weekPlans, DayMealPlan{
			Day:           day,
			Date:          time.Now().AddDate(0, 0, day-1),
			BreakfastDish: breakfastDish,
			LunchDish:     lunchDish,
			DinnerDish:    dinnerDish,
			TotalCalories: dayCalories,
			TotalProtein:  dayProtein,
			TotalCarbs:    dayCarbs,
			TotalFat:      dayFat,
		})

		weekCalories += dayCalories
		weekProtein += dayProtein
		weekCarbs += dayCarbs
		weekFat += dayFat
	}

	return c.Status(http.StatusOK).JSON(WeekMealPlanResponse{
		SessionID:      req.SessionID,
		Days:           req.Days,
		MealPlans:      weekPlans,
		WeekCalories:   weekCalories,
		WeekProtein:    weekProtein,
		WeekCarbs:      weekCarbs,
		WeekFat:        weekFat,
		TargetCalories: session.DailyCalorieTarget * req.Days,
	})
}

// SaveMealPlan persists the meal plan selection to the database
func SaveMealPlan(c *fiber.Ctx) error {
	var req MealPlanSaveRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.SessionID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Session ID is required",
		})
	}

	// Get session to check targets
	var session models.Session
	if err := db.DB.First(&session, "id = ?", req.SessionID).Error; err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Session not found",
		})
	}

	// Get the dishes and calculate totals
	mealPlan := models.MealPlan{
		SessionID:       session.ID,
		MealDate:        time.Now(),
		BreakfastDishID: req.BreakfastDishID,
		LunchDishID:     req.LunchDishID,
		DinnerDishID:    req.DinnerDishID,
	}

	// Fetch dishes and calculate totals
	if req.BreakfastDishID != nil {
		var dish models.Dish
		if err := db.DB.First(&dish, "id = ?", *req.BreakfastDishID).Error; err == nil {
			mealPlan.TotalCalories += dish.Calories
			mealPlan.TotalProtein += dish.Protein
			mealPlan.TotalCarbs += dish.Carbs
			mealPlan.TotalFat += dish.Fat
		}
	}

	if req.LunchDishID != nil {
		var dish models.Dish
		if err := db.DB.First(&dish, "id = ?", *req.LunchDishID).Error; err == nil {
			mealPlan.TotalCalories += dish.Calories
			mealPlan.TotalProtein += dish.Protein
			mealPlan.TotalCarbs += dish.Carbs
			mealPlan.TotalFat += dish.Fat
		}
	}

	if req.DinnerDishID != nil {
		var dish models.Dish
		if err := db.DB.First(&dish, "id = ?", *req.DinnerDishID).Error; err == nil {
			mealPlan.TotalCalories += dish.Calories
			mealPlan.TotalProtein += dish.Protein
			mealPlan.TotalCarbs += dish.Carbs
			mealPlan.TotalFat += dish.Fat
		}
	}

	// Validate totals are within 10% of target
	calorieDeviation := float64(mealPlan.TotalCalories-session.DailyCalorieTarget) / float64(session.DailyCalorieTarget)
	if calorieDeviation > 0.1 || calorieDeviation < -0.1 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error":             fmt.Sprintf("Meal plan total calories (%d) deviates more than 10%% from target (%d)", mealPlan.TotalCalories, session.DailyCalorieTarget),
			"total_calories":    mealPlan.TotalCalories,
			"target_calories":   session.DailyCalorieTarget,
			"deviation_percent": calorieDeviation * 100,
		})
	}

	// Save to database
	if err := db.DB.Create(&mealPlan).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save meal plan",
		})
	}

	return c.Status(http.StatusCreated).JSON(mealPlan)
}

// GetMealPlan retrieves the latest meal plan for a session
func GetMealPlan(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Session ID is required",
		})
	}

	var mealPlan models.MealPlan
	if err := db.DB.
		Preload("BreakfastDish").
		Preload("LunchDish").
		Preload("DinnerDish").
		Where("session_id = ?", sessionID).
		Order("meal_date DESC").
		First(&mealPlan).Error; err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Meal plan not found for session",
		})
	}

	return c.Status(http.StatusOK).JSON(mealPlan)
}

// getDishForMeal fetches a random dish matching the meal type and diet type within calorie range
// Calorie ranges: Breakfast ~25% of daily, Lunch ~35% of daily, Dinner ~30% of daily
func getDishForMeal(mealType string, dietType string, dailyCalories int) (*models.Dish, error) {
	var minCal, maxCal int

	// Define calorie ranges for each meal
	switch mealType {
	case "Breakfast":
		minCal = int(float64(dailyCalories) * 0.20) // 20% of daily
		maxCal = int(float64(dailyCalories) * 0.30) // 30% of daily
	case "Lunch":
		minCal = int(float64(dailyCalories) * 0.32) // 32% of daily
		maxCal = int(float64(dailyCalories) * 0.40) // 40% of daily
	case "Dinner":
		minCal = int(float64(dailyCalories) * 0.28) // 28% of daily
		maxCal = int(float64(dailyCalories) * 0.36) // 36% of daily
	default:
		minCal = int(float64(dailyCalories) * 0.25)
		maxCal = int(float64(dailyCalories) * 0.35)
	}

	var dish models.Dish

	// Try 1: Get dish matching meal_type, diet_type, and calorie range
	if err := db.DB.
		Where("meal_type = ?", mealType).
		Where("diet_type = ?", dietType).
		Where("calories BETWEEN ? AND ?", minCal, maxCal).
		Order("RANDOM()").
		First(&dish).Error; err == nil {
		return &dish, nil
	}

	// Try 2: Expand the calorie range
	minCal = int(float64(dailyCalories) * 0.15)
	maxCal = int(float64(dailyCalories) * 0.45)
	if err := db.DB.
		Where("meal_type = ?", mealType).
		Where("diet_type = ?", dietType).
		Where("calories BETWEEN ? AND ?", minCal, maxCal).
		Order("RANDOM()").
		First(&dish).Error; err == nil {
		return &dish, nil
	}

	// Try 3: Any dish of meal_type with diet_type (ignore calories)
	if err := db.DB.
		Where("meal_type = ?", mealType).
		Where("diet_type = ?", dietType).
		Order("RANDOM()").
		First(&dish).Error; err == nil {
		return &dish, nil
	}

	// Try 4: Any dish of meal_type regardless of diet type (fallback)
	if err := db.DB.
		Where("meal_type = ?", mealType).
		Order("RANDOM()").
		First(&dish).Error; err == nil {
		return &dish, nil
	}

	// Try 5: Last resort - get any dish
	if err := db.DB.Order("RANDOM()").First(&dish).Error; err == nil {
		return &dish, nil
	}

	return nil, fmt.Errorf("no dishes available in database")
}

// getDishForMealWithExclusion fetches a dish while excluding already used dishes
func getDishForMealWithExclusion(mealType string, dietType string, dailyCalories int, usedDishIDs map[uint]bool) (*models.Dish, error) {
	var minCal, maxCal int

	// Define calorie ranges for each meal
	switch mealType {
	case "Breakfast":
		minCal = int(float64(dailyCalories) * 0.20) // 20% of daily
		maxCal = int(float64(dailyCalories) * 0.30) // 30% of daily
	case "Lunch":
		minCal = int(float64(dailyCalories) * 0.32) // 32% of daily
		maxCal = int(float64(dailyCalories) * 0.40) // 40% of daily
	case "Dinner":
		minCal = int(float64(dailyCalories) * 0.28) // 28% of daily
		maxCal = int(float64(dailyCalories) * 0.36) // 36% of daily
	default:
		minCal = int(float64(dailyCalories) * 0.25)
		maxCal = int(float64(dailyCalories) * 0.35)
	}

	var dish models.Dish
	var excludedIDs []uint
	for id := range usedDishIDs {
		excludedIDs = append(excludedIDs, id)
	}

	// Try 1: Get dish matching meal_type, diet_type, and calorie range (excluding used dishes)
	if len(excludedIDs) > 0 {
		if err := db.DB.
			Where("meal_type = ?", mealType).
			Where("diet_type = ?", dietType).
			Where("calories BETWEEN ? AND ?", minCal, maxCal).
			Where("id NOT IN ?", excludedIDs).
			Order("RANDOM()").
			First(&dish).Error; err == nil {
			return &dish, nil
		}
	} else {
		if err := db.DB.
			Where("meal_type = ?", mealType).
			Where("diet_type = ?", dietType).
			Where("calories BETWEEN ? AND ?", minCal, maxCal).
			Order("RANDOM()").
			First(&dish).Error; err == nil {
			return &dish, nil
		}
	}

	// Try 2: Expand the calorie range (excluding used dishes)
	minCal = int(float64(dailyCalories) * 0.15)
	maxCal = int(float64(dailyCalories) * 0.45)
	if len(excludedIDs) > 0 {
		if err := db.DB.
			Where("meal_type = ?", mealType).
			Where("diet_type = ?", dietType).
			Where("calories BETWEEN ? AND ?", minCal, maxCal).
			Where("id NOT IN ?", excludedIDs).
			Order("RANDOM()").
			First(&dish).Error; err == nil {
			return &dish, nil
		}
	} else {
		if err := db.DB.
			Where("meal_type = ?", mealType).
			Where("diet_type = ?", dietType).
			Where("calories BETWEEN ? AND ?", minCal, maxCal).
			Order("RANDOM()").
			First(&dish).Error; err == nil {
			return &dish, nil
		}
	}

	// Try 3: Any dish of meal_type with diet_type (excluding used dishes)
	if len(excludedIDs) > 0 {
		if err := db.DB.
			Where("meal_type = ?", mealType).
			Where("diet_type = ?", dietType).
			Where("id NOT IN ?", excludedIDs).
			Order("RANDOM()").
			First(&dish).Error; err == nil {
			return &dish, nil
		}
	} else {
		if err := db.DB.
			Where("meal_type = ?", mealType).
			Where("diet_type = ?", dietType).
			Order("RANDOM()").
			First(&dish).Error; err == nil {
			return &dish, nil
		}
	}

	// Try 4: Any dish of meal_type regardless of diet type (excluding used dishes)
	if len(excludedIDs) > 0 {
		if err := db.DB.
			Where("meal_type = ?", mealType).
			Where("id NOT IN ?", excludedIDs).
			Order("RANDOM()").
			First(&dish).Error; err == nil {
			return &dish, nil
		}
	} else {
		if err := db.DB.
			Where("meal_type = ?", mealType).
			Order("RANDOM()").
			First(&dish).Error; err == nil {
			return &dish, nil
		}
	}

	// Try 5: Last resort - get any dish (excluding used)
	if len(excludedIDs) > 0 {
		if err := db.DB.
			Where("id NOT IN ?", excludedIDs).
			Order("RANDOM()").
			First(&dish).Error; err == nil {
			return &dish, nil
		}
	} else {
		if err := db.DB.Order("RANDOM()").First(&dish).Error; err == nil {
			return &dish, nil
		}
	}

	return nil, fmt.Errorf("no dishes available in database")
}

// Structs for 7-day meal plan response
type DayMealPlan struct {
	Day           int          `json:"day"`
	Date          time.Time    `json:"date"`
	BreakfastDish *models.Dish `json:"breakfast_dish"`
	LunchDish     *models.Dish `json:"lunch_dish"`
	DinnerDish    *models.Dish `json:"dinner_dish"`
	TotalCalories int          `json:"total_calories"`
	TotalProtein  float64      `json:"total_protein"`
	TotalCarbs    float64      `json:"total_carbs"`
	TotalFat      float64      `json:"total_fat"`
}

type WeekMealPlanResponse struct {
	SessionID      string        `json:"session_id"`
	Days           int           `json:"days"`
	MealPlans      []DayMealPlan `json:"meal_plans"`
	WeekCalories   int           `json:"week_calories"`
	WeekProtein    float64       `json:"week_protein"`
	WeekCarbs      float64       `json:"week_carbs"`
	WeekFat        float64       `json:"week_fat"`
	TargetCalories int           `json:"target_calories"`
}

type SuggestWeekMealPlanRequest struct {
	SessionID   string `json:"session_id"`
	FitnessGoal string `json:"fitness_goal"`
	DietType    string `json:"diet_type"`
	Days        int    `json:"days"` // 1-7 days
}

type ExportMealPlanRequest struct {
	BreakfastDishID *uint `json:"breakfast_dish_id"`
	LunchDishID     *uint `json:"lunch_dish_id"`
	DinnerDishID    *uint `json:"dinner_dish_id"`
}

type WeekMealData struct {
	Day             int   `json:"day"`
	BreakfastDishID *uint `json:"breakfast_dish_id"`
	LunchDishID     *uint `json:"lunch_dish_id"`
	DinnerDishID    *uint `json:"dinner_dish_id"`
}

type ExportWeekMealPlanRequest struct {
	SessionID string         `json:"session_id"`
	WeekMeals []WeekMealData `json:"week_meals"`
}

// ExportMealPlanPDF generates and returns a PDF file of the meal plan
func ExportMealPlanPDF(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	if sessionID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Session ID is required",
		})
	}

	// Get session
	var session models.Session
	if err := db.DB.First(&session, "id = ?", sessionID).Error; err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Session not found",
		})
	}

	// Try to get meal plan from database first
	var mealPlan models.MealPlan
	dbErr := db.DB.
		Preload("BreakfastDish").
		Preload("LunchDish").
		Preload("DinnerDish").
		Where("session_id = ?", sessionID).
		Order("meal_date DESC").
		First(&mealPlan).Error

	// If meal plan doesn't exist in DB, try to get it from request body
	if dbErr != nil {
		var exportReq ExportMealPlanRequest
		if err := c.BodyParser(&exportReq); err != nil || (exportReq.BreakfastDishID == nil && exportReq.LunchDishID == nil && exportReq.DinnerDishID == nil) {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error": "Meal plan not found and no meal data provided. Please save the meal plan first or provide meal data in request body.",
			})
		}

		// Build meal plan from request data
		mealPlan.SessionID = sessionID
		mealPlan.MealDate = time.Now()

		// Load the dishes
		if exportReq.BreakfastDishID != nil {
			var breakfastDish models.Dish
			if err := db.DB.First(&breakfastDish, "id = ?", *exportReq.BreakfastDishID).Error; err == nil {
				mealPlan.BreakfastDish = &breakfastDish
				mealPlan.BreakfastDishID = exportReq.BreakfastDishID
			}
		}

		if exportReq.LunchDishID != nil {
			var lunchDish models.Dish
			if err := db.DB.First(&lunchDish, "id = ?", *exportReq.LunchDishID).Error; err == nil {
				mealPlan.LunchDish = &lunchDish
				mealPlan.LunchDishID = exportReq.LunchDishID
			}
		}

		if exportReq.DinnerDishID != nil {
			var dinnerDish models.Dish
			if err := db.DB.First(&dinnerDish, "id = ?", *exportReq.DinnerDishID).Error; err == nil {
				mealPlan.DinnerDish = &dinnerDish
				mealPlan.DinnerDishID = exportReq.DinnerDishID
			}
		}

		// Calculate totals
		if mealPlan.BreakfastDish != nil {
			mealPlan.TotalCalories += mealPlan.BreakfastDish.Calories
			mealPlan.TotalProtein += mealPlan.BreakfastDish.Protein
			mealPlan.TotalCarbs += mealPlan.BreakfastDish.Carbs
			mealPlan.TotalFat += mealPlan.BreakfastDish.Fat
		}
		if mealPlan.LunchDish != nil {
			mealPlan.TotalCalories += mealPlan.LunchDish.Calories
			mealPlan.TotalProtein += mealPlan.LunchDish.Protein
			mealPlan.TotalCarbs += mealPlan.LunchDish.Carbs
			mealPlan.TotalFat += mealPlan.LunchDish.Fat
		}
		if mealPlan.DinnerDish != nil {
			mealPlan.TotalCalories += mealPlan.DinnerDish.Calories
			mealPlan.TotalProtein += mealPlan.DinnerDish.Protein
			mealPlan.TotalCarbs += mealPlan.DinnerDish.Carbs
			mealPlan.TotalFat += mealPlan.DinnerDish.Fat
		}
	}

	// Generate PDF
	pdfService := services.NewPDFService()
	pdfBytes, err := pdfService.GenerateMealPlanPDF(&session, &mealPlan)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to generate PDF: %v", err),
		})
	}

	// Set response headers
	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"meal-plan-%s.pdf\"", time.Now().Format("2006-01-02")))

	return c.Send(pdfBytes)
}

// ExportWeekMealPlanPDF generates a PDF for the entire week meal plan with full recipe details
func ExportWeekMealPlanPDF(c *fiber.Ctx) error {
	var req ExportWeekMealPlanRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.SessionID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Session ID is required",
		})
	}

	// Get session
	var session models.Session
	if err := db.DB.First(&session, "id = ?", req.SessionID).Error; err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Session not found",
		})
	}

	// Build week meal plans with full dish details
	type DayMealDetail struct {
		Day           int          `json:"day"`
		BreakfastDish *models.Dish `json:"breakfast_dish"`
		LunchDish     *models.Dish `json:"lunch_dish"`
		DinnerDish    *models.Dish `json:"dinner_dish"`
	}

	var weekPlans []DayMealDetail

	for _, dayData := range req.WeekMeals {
		dayPlan := DayMealDetail{Day: dayData.Day}

		if dayData.BreakfastDishID != nil {
			var dish models.Dish
			if err := db.DB.First(&dish, "id = ?", *dayData.BreakfastDishID).Error; err == nil {
				dayPlan.BreakfastDish = &dish
			}
		}

		if dayData.LunchDishID != nil {
			var dish models.Dish
			if err := db.DB.First(&dish, "id = ?", *dayData.LunchDishID).Error; err == nil {
				dayPlan.LunchDish = &dish
			}
		}

		if dayData.DinnerDishID != nil {
			var dish models.Dish
			if err := db.DB.First(&dish, "id = ?", *dayData.DinnerDishID).Error; err == nil {
				dayPlan.DinnerDish = &dish
			}
		}

		weekPlans = append(weekPlans, dayPlan)
	}

	// Generate PDF with full recipe details
	pdfService := services.NewPDFService()
	pdfBytes, err := pdfService.GenerateWeekMealPlanPDF(&session, weekPlans)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to generate PDF: %v", err),
		})
	}

	// Set response headers
	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"week-meal-plan-%s.pdf\"", time.Now().Format("2006-01-02")))

	return c.Send(pdfBytes)
}

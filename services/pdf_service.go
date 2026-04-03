package services

import (
	"fitfuel/models"
	"fmt"
	"reflect"
	"strings"

	"github.com/jung-kurt/gofpdf"
)

type PDFService struct {
	pdf *gofpdf.Fpdf
}

func NewPDFService() *PDFService {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "", 12)
	return &PDFService{pdf: pdf}
}

// GenerateMealPlanPDF creates a PDF document for the meal plan
func (ps *PDFService) GenerateMealPlanPDF(session *models.Session, mealPlan *models.MealPlan) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Title
	pdf.SetFont("Arial", "B", 20)
	pdf.CellFormat(0, 15, "Your Daily Meal Plan", "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "", 11)
	pdf.CellFormat(0, 5, "Generated on "+mealPlan.MealDate.Format("02 Jan 2006"), "", 1, "C", false, 0, "")
	pdf.Ln(5)

	// User Information Section
	pdf.SetFont("Arial", "B", 13)
	pdf.CellFormat(0, 8, "Your Profile", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 11)

	ProfileData := []struct{ key, value string }{
		{"Weight", fmt.Sprintf("%.1f kg", session.Weight)},
		{"Height", fmt.Sprintf("%.1f cm", session.Height)},
		{"Age", fmt.Sprintf("%d years", session.Age)},
		{"Gender", session.Gender},
		{"Fitness Goal", session.FitnessGoal},
		{"Diet Type", session.DietType},
	}

	if session.TargetWeight != nil {
		ProfileData = append(ProfileData, struct{ key, value string }{"Target Weight", fmt.Sprintf("%.1f kg", *session.TargetWeight)})
	}

	for _, item := range ProfileData {
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(50, 6, item.key+":", "", 0, "L", false, 0, "")
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(0, 6, item.value, "", 1, "L", false, 0, "")
	}
	pdf.Ln(5)

	// Daily Nutrition Targets
	pdf.SetFont("Arial", "B", 13)
	pdf.CellFormat(0, 8, "Daily Nutrition Targets", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 10)

	// Create table for targets
	targets := [][]string{
		{"Nutrient", "Target", "Your Plan", "Match"},
		{
			"Calories",
			fmt.Sprintf("%d kcal", session.DailyCalorieTarget),
			fmt.Sprintf("%d kcal", mealPlan.TotalCalories),
			fmt.Sprintf("%.1f%%", (float64(mealPlan.TotalCalories)/float64(session.DailyCalorieTarget))*100),
		},
		{
			"Protein",
			fmt.Sprintf("%.1f g", session.DailyProteinTarget),
			fmt.Sprintf("%.1f g", mealPlan.TotalProtein),
			fmt.Sprintf("%.1f%%", (mealPlan.TotalProtein/session.DailyProteinTarget)*100),
		},
		{
			"Carbohydrates",
			fmt.Sprintf("%.1f g", session.DailyCarbsTarget),
			fmt.Sprintf("%.1f g", mealPlan.TotalCarbs),
			fmt.Sprintf("%.1f%%", (mealPlan.TotalCarbs/session.DailyCarbsTarget)*100),
		},
		{
			"Fat",
			fmt.Sprintf("%.1f g", session.DailyFatTarget),
			fmt.Sprintf("%.1f g", mealPlan.TotalFat),
			fmt.Sprintf("%.1f%%", (mealPlan.TotalFat/session.DailyFatTarget)*100),
		},
	}

	drawTable(pdf, targets)
	pdf.Ln(5)

	// Meals Section
	pdf.SetFont("Arial", "B", 13)
	pdf.CellFormat(0, 8, "Your Meals", "", 1, "L", false, 0, "")
	pdf.Ln(3)

	if mealPlan.BreakfastDish != nil {
		drawMealSection(pdf, "Breakfast", mealPlan.BreakfastDish)
	}

	if mealPlan.LunchDish != nil {
		drawMealSection(pdf, "Lunch", mealPlan.LunchDish)
	}

	if mealPlan.DinnerDish != nil {
		drawMealSection(pdf, "Dinner", mealPlan.DinnerDish)
	}

	// Shopping List
	pdf.SetFont("Arial", "B", 13)
	pdf.CellFormat(0, 8, "Shopping List", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 11)

	ingredientList := collectIngredients(mealPlan)
	for _, ingredient := range ingredientList {
		pdf.CellFormat(5, 6, "- ", "", 0, "L", false, 0, "")
		pdf.MultiCell(0, 6, ingredient, "", "L", false)
	}

	// Return PDF as bytes
	var buf strings.Builder
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}

	return []byte(buf.String()), nil
}

func drawMealSection(pdf *gofpdf.Fpdf, mealName string, dish *models.Dish) {
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(0, 7, mealName+": "+dish.Name, "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 10)

	mealInfo := []string{
		fmt.Sprintf("Calories: %d kcal", dish.Calories),
		fmt.Sprintf("Protein: %.1f g | Carbs: %.1f g | Fat: %.1f g", dish.Protein, dish.Carbs, dish.Fat),
	}

	if dish.PrepTime != "" {
		mealInfo = append(mealInfo, fmt.Sprintf("Prep Time: %s", dish.PrepTime))
	}

	if dish.Difficulty != "" {
		mealInfo = append(mealInfo, fmt.Sprintf("Difficulty: %s", dish.Difficulty))
	}

	for _, info := range mealInfo {
		pdf.CellFormat(0, 5, info, "", 1, "L", false, 0, "")
	}

	// Recipe Steps
	if len(dish.Steps) > 0 {
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(0, 5, "Cooking Instructions:", "", 1, "L", false, 0, "")
		pdf.SetFont("Arial", "", 9)

		stepsStr := string(dish.Steps)
		stepsStr = strings.Trim(stepsStr, "[]\"")
		steps := strings.Split(stepsStr, "|")
		for i, step := range steps {
			step = strings.TrimSpace(step)
			step = strings.Trim(step, "\"")
			if step != "" {
				pdf.CellFormat(15, 5, fmt.Sprintf("Step %d: ", i+1), "", 0, "L", false, 0, "")
				pdf.MultiCell(0, 5, step, "", "L", false)
			}
		}
	}

	// Ingredients
	if len(dish.Ingredients) > 0 {
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(0, 5, "Key Ingredients:", "", 1, "L", false, 0, "")
		pdf.SetFont("Arial", "", 9)

		var ingredientList []string
		if err := dish.Ingredients.UnmarshalJSON(dish.Ingredients); err == nil {
			ingredientList = append(ingredientList, string(dish.Ingredients))
		} else {
			ingredientStr := string(dish.Ingredients)
			ingredientStr = strings.Trim(ingredientStr, "[]\"")
			ingredientList = strings.Split(ingredientStr, ",")
		}

		for _, ing := range ingredientList {
			ing = strings.TrimSpace(ing)
			ing = strings.Trim(ing, "\"")
			if ing != "" {
				pdf.CellFormat(5, 5, "• ", "", 0, "L", false, 0, "")
				pdf.MultiCell(0, 5, ing, "", "L", false)
			}
		}
	}

	pdf.Ln(3)
}

func collectIngredients(mealPlan *models.MealPlan) []string {
	ingredients := []string{}
	seen := make(map[string]bool)

	if mealPlan.BreakfastDish != nil {
		ingredients = append(ingredients, extractIngredients(mealPlan.BreakfastDish, seen)...)
	}
	if mealPlan.LunchDish != nil {
		ingredients = append(ingredients, extractIngredients(mealPlan.LunchDish, seen)...)
	}
	if mealPlan.DinnerDish != nil {
		ingredients = append(ingredients, extractIngredients(mealPlan.DinnerDish, seen)...)
	}

	return ingredients
}

func extractIngredients(dish *models.Dish, seen map[string]bool) []string {
	var ingredients []string

	var ingredientList []string
	if err := dish.Ingredients.UnmarshalJSON(dish.Ingredients); err == nil {
		// Successfully unmarshaled, use the ingredients
		for _, ing := range ingredientList {
			ingLower := strings.ToLower(strings.TrimSpace(ing))
			if !seen[ingLower] {
				ingredients = append(ingredients, ing)
				seen[ingLower] = true
			}
		}
	}

	return ingredients
}

func drawTable(pdf *gofpdf.Fpdf, data [][]string) {
	// Column widths
	colWidths := []float64{45, 45, 45, 35}
	lineHeight := 7.0

	// Header
	pdf.SetFont("Arial", "B", 10)
	for i, header := range data[0] {
		pdf.CellFormat(colWidths[i], lineHeight, header, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(lineHeight)

	// Data rows
	pdf.SetFont("Arial", "", 9)
	for _, row := range data[1:] {
		for i, cell := range row {
			pdf.CellFormat(colWidths[i], lineHeight, cell, "1", 0, "C", false, 0, "")
		}
		pdf.Ln(lineHeight)
	}
}

// GenerateWeekMealPlanPDF creates a comprehensive PDF for the entire week meal plan with full recipe details
func (ps *PDFService) GenerateWeekMealPlanPDF(session *models.Session, weekPlans interface{}) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")

	// Title Page
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 24)
	pdf.SetTextColor(51, 51, 51)
	pdf.CellFormat(0, 20, "Your Weekly Meal Plan", "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "", 12)
	pdf.SetTextColor(100, 100, 100)
	pdf.CellFormat(0, 8, "Generated on "+fmt.Sprintf("%02d %s %d", 2, "Apr", 2026), "", 1, "C", false, 0, "")
	pdf.Ln(10)

	// User Profile Section
	pdf.SetFont("Arial", "B", 14)
	pdf.SetTextColor(51, 51, 51)
	pdf.CellFormat(0, 8, "Your Profile", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(80, 80, 80)

	profileTable := [][]string{
		{"Weight", fmt.Sprintf("%.1f kg", session.Weight), "Height", fmt.Sprintf("%.1f cm", session.Height)},
		{"Age", fmt.Sprintf("%d years", session.Age), "Gender", session.Gender},
		{"Fitness Goal", session.FitnessGoal, "Diet Type", session.DietType},
	}

	if session.TargetWeight != nil {
		profileTable = append(profileTable, []string{"Target Weight", fmt.Sprintf("%.1f kg", *session.TargetWeight), "", ""})
	}

	// Draw simple profile layout
	for _, row := range profileTable {
		pdf.CellFormat(40, 6, row[0]+":", "", 0, "L", false, 0, "")
		pdf.CellFormat(50, 6, row[1], "", 0, "L", false, 0, "")
		pdf.CellFormat(40, 6, row[2]+":", "", 0, "L", false, 0, "")
		pdf.CellFormat(0, 6, row[3], "", 1, "L", false, 0, "")
	}
	pdf.Ln(5)

	// Daily Nutrition Targets Section
	pdf.SetFont("Arial", "B", 14)
	pdf.SetTextColor(51, 51, 51)
	pdf.CellFormat(0, 8, "Daily Nutrition Targets", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(80, 80, 80)

	targets := [][]string{
		{"Nutrient", "Target", "Daily Goal"},
		{"Calories", fmt.Sprintf("%d kcal", session.DailyCalorieTarget), "Energy intake"},
		{"Protein", fmt.Sprintf("%.1f g", session.DailyProteinTarget), "Muscle building"},
		{"Carbohydrates", fmt.Sprintf("%.1f g", session.DailyCarbsTarget), "Energy source"},
		{"Fat", fmt.Sprintf("%.1f g", session.DailyFatTarget), "Essential nutrients"},
	}

	drawTableWithColWidths(pdf, targets, []float64{50, 50, 80})
	pdf.Ln(8)

	// Weekly Meal Plans Section
	pdf.SetFont("Arial", "B", 14)
	pdf.SetTextColor(51, 51, 51)
	pdf.CellFormat(0, 8, "Your Weekly Meal Plans", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(80, 80, 80)
	pdf.CellFormat(0, 5, "Complete recipe details for each meal throughout the week", "", 1, "L", false, 0, "")
	pdf.Ln(3)

	// Process day plans with reflection
	rv := reflect.ValueOf(weekPlans)
	if rv.Kind() == reflect.Slice && rv.Len() > 0 {
		for i := 0; i < rv.Len(); i++ {
			dayPlanVal := rv.Index(i)
			dayField := dayPlanVal.FieldByName("Day")
			if !dayField.IsValid() {
				continue
			}
			day := int(dayField.Int())

			// Day header with separator
			pdf.SetDrawColor(200, 200, 200)
			pdf.CellFormat(0, 0.5, "", "", 1, "L", false, 0, "") // Horizontal line
			pdf.Ln(2)

			pdf.SetFont("Arial", "B", 12)
			pdf.SetTextColor(80, 100, 200)
			pdf.CellFormat(0, 7, fmt.Sprintf("Day %d", day), "", 1, "L", false, 0, "")
			pdf.SetTextColor(80, 80, 80)
			pdf.Ln(2)

			// Breakfast
			breakfastField := dayPlanVal.FieldByName("BreakfastDish")
			if breakfastField.IsValid() && !breakfastField.IsNil() {
				if dish, ok := breakfastField.Interface().(*models.Dish); ok {
					drawMealSectionCompact(pdf, "Breakfast", dish)
				}
			}

			// Lunch
			lunchField := dayPlanVal.FieldByName("LunchDish")
			if lunchField.IsValid() && !lunchField.IsNil() {
				if dish, ok := lunchField.Interface().(*models.Dish); ok {
					drawMealSectionCompact(pdf, "Lunch", dish)
				}
			}

			// Dinner
			dinnerField := dayPlanVal.FieldByName("DinnerDish")
			if dinnerField.IsValid() && !dinnerField.IsNil() {
				if dish, ok := dinnerField.Interface().(*models.Dish); ok {
					drawMealSectionCompact(pdf, "Dinner", dish)
				}
			}

			pdf.Ln(3)

			// Auto page break
			if pdf.GetY() > 250 {
				pdf.AddPage()
				pdf.SetFont("Arial", "", 10)
				pdf.SetTextColor(80, 80, 80)
			}
		}
	}

	// Return PDF as bytes
	var buf strings.Builder
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}

	return []byte(buf.String()), nil
}

func drawTableWithColWidths(pdf *gofpdf.Fpdf, data [][]string, colWidths []float64) {
	lineHeight := 7.0

	// Header
	pdf.SetFont("Arial", "B", 10)
	for i, header := range data[0] {
		if i < len(colWidths) {
			pdf.CellFormat(colWidths[i], lineHeight, header, "1", 0, "C", false, 0, "")
		}
	}
	pdf.Ln(lineHeight)

	// Data rows
	pdf.SetFont("Arial", "", 9)
	for _, row := range data[1:] {
		for i, cell := range row {
			if i < len(colWidths) {
				pdf.CellFormat(colWidths[i], lineHeight, cell, "1", 0, "L", false, 0, "")
			}
		}
		pdf.Ln(lineHeight)
	}
}

// drawMealSectionCompact is a more compact version for weekly PDFs
func drawMealSectionCompact(pdf *gofpdf.Fpdf, mealName string, dish *models.Dish) {
	pdf.SetFont("Arial", "B", 11)
	pdf.SetTextColor(51, 51, 51)
	pdf.CellFormat(0, 6, mealName+": "+dish.Name, "", 1, "L", false, 0, "")

	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(100, 100, 100)

	// Basic nutrition info on one line
	infoText := fmt.Sprintf("Calories: %d | Protein: %.1f g | Carbs: %.1f g | Fat: %.1f g",
		dish.Calories, dish.Protein, dish.Carbs, dish.Fat)
	pdf.CellFormat(0, 4, infoText, "", 1, "L", false, 0, "")

	// Difficulty and prep time on same line
	if dish.Difficulty != "" || dish.PrepTime != "" {
		metaText := ""
		if dish.PrepTime != "" {
			metaText = "Prep: " + dish.PrepTime
		}
		if dish.Difficulty != "" {
			if metaText != "" {
				metaText += " | "
			}
			metaText += "Difficulty: " + dish.Difficulty
		}
		pdf.SetTextColor(120, 120, 120)
		pdf.CellFormat(0, 4, metaText, "", 1, "L", false, 0, "")
	}

	// Recipe Steps - labeled as "Instructions"
	if len(dish.Steps) > 0 {
		pdf.Ln(1)
		pdf.SetFont("Arial", "B", 9)
		pdf.SetTextColor(51, 51, 51)
		pdf.CellFormat(0, 4, "Instructions:", "", 1, "L", false, 0, "")
		pdf.SetFont("Arial", "", 8)
		pdf.SetTextColor(80, 80, 80)

		stepsStr := string(dish.Steps)
		stepsStr = strings.Trim(stepsStr, "[]\"")
		steps := strings.Split(stepsStr, "|")
		for i, step := range steps {
			step = strings.TrimSpace(step)
			step = strings.Trim(step, "\"")
			if step != "" && step != " " {
				stepText := fmt.Sprintf("  %d. %s", i+1, step)
				pdf.MultiCell(0, 4, stepText, "", "L", false)
			}
		}
	}

	// Ingredients section - properly parse JSON
	if len(dish.Ingredients) > 0 {
		pdf.Ln(1)
		pdf.SetFont("Arial", "B", 9)
		pdf.SetTextColor(51, 51, 51)
		pdf.CellFormat(0, 4, "Ingredients:", "", 1, "L", false, 0, "")
		pdf.SetFont("Arial", "", 8)
		pdf.SetTextColor(80, 80, 80)

		ingredientList := parseIngredientsJSON(dish.Ingredients)

		for _, ing := range ingredientList {
			ing = strings.TrimSpace(ing)
			ing = strings.Trim(ing, "\"")
			if ing != "" && ing != " " {
				pdf.CellFormat(5, 5, "-", "", 0, "L", false, 0, "")
				pdf.MultiCell(0, 4, " "+ing, "", "L", false)
			}
		}
	}

	pdf.Ln(2)
	pdf.SetTextColor(0, 0, 0)
}

// parseIngredientsJSON parses JSON array format ingredients
func parseIngredientsJSON(jsonData []byte) []string {
	var ingredients []string

	// Convert to string and clean up
	str := string(jsonData)
	str = strings.Trim(str, "[]")

	// Split by comma - but be careful with quoted strings
	parts := strings.Split(str, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, "\"")
		if part != "" {
			ingredients = append(ingredients, part)
		}
	}

	return ingredients
}

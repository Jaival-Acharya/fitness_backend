package handlers

import (
	"encoding/json"
	"fitfuel/db"
	"fitfuel/models"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func GetDishes(c *fiber.Ctx) error {
	diet := c.Query("diet")
	meal := c.Query("meal")
	excludeAllergens := parseAllergenList(c.Query("exclude_allergens"))
	sessionID := c.Query("session_id")
	if sessionID != "" {
		var session models.Session
		if err := db.DB.First(&session, "id = ?", sessionID).Error; err == nil {
			stored := parseAllergiesJSON(session.Allergies)
			excludeAllergens = mergeAllergens(excludeAllergens, stored)
		}
	}

	if diet == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Diet type is required",
		})
	}

	var dishes []models.Dish
	query := db.DB

	// Filter by diet type
	// nonveg sees all dishes
	// egg sees veg + egg dishes
	// veg sees only veg dishes
	switch diet {
	case "nonveg":
		// All dishes
	case "egg":
		query = query.Where("diet_type IN ?", []string{"veg", "egg"})
	case "veg":
		query = query.Where("diet_type = ?", "veg")
	default:
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid diet type",
		})
	}

	// Filter by meal type if provided
	if meal != "" {
		query = query.Where("meal_type = ?", meal)
	}

	if err := query.Find(&dishes).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch dishes",
		})
	}

	if len(excludeAllergens) > 0 {
		filtered := make([]models.Dish, 0, len(dishes))
		for _, dish := range dishes {
			if shouldExcludeDish(dish, excludeAllergens) {
				continue
			}
			filtered = append(filtered, dish)
		}
		dishes = filtered
	}

	return c.Status(http.StatusOK).JSON(dishes)
}

func GetDishByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Dish ID is required",
		})
	}

	var dish models.Dish
	if err := db.DB.First(&dish, id).Error; err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Dish not found",
		})
	}

	return c.Status(http.StatusOK).JSON(dish)
}

func parseAllergenList(raw string) []string {
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	seen := make(map[string]struct{}, len(parts))
	allergens := make([]string, 0, len(parts))

	for _, part := range parts {
		item := strings.ToLower(strings.TrimSpace(part))
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		allergens = append(allergens, item)
	}

	return allergens
}

func parseAllergiesJSON(raw []byte) []string {
	if len(raw) == 0 {
		return nil
	}

	var allergens []string
	if err := json.Unmarshal(raw, &allergens); err != nil {
		return nil
	}

	return parseAllergenList(strings.Join(allergens, ","))
}

func mergeAllergens(base []string, extra []string) []string {
	seen := make(map[string]struct{}, len(base)+len(extra))
	out := make([]string, 0, len(base)+len(extra))

	for _, item := range base {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}

	for _, item := range extra {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}

	return out
}

func shouldExcludeDish(dish models.Dish, allergens []string) bool {
	tags := parseAllergiesJSON(dish.AllergenTags)
	tagSet := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tagSet[tag] = struct{}{}
	}

	ingredientText := strings.ToLower(string(dish.Ingredients))

	for _, allergen := range allergens {
		if _, ok := tagSet[allergen]; ok {
			return true
		}

		for _, keyword := range allergenKeywords(allergen) {
			if strings.Contains(ingredientText, keyword) {
				return true
			}
		}
	}

	return false
}

func allergenKeywords(allergen string) []string {
	switch allergen {
	case "milk":
		return []string{"milk", "dairy", "paneer", "cheese", "yoghurt", "yogurt", "cream", "butter", "ghee", "mayo", "mayonnaise"}
	case "egg":
		return []string{"egg", "eggs", "omelette", "omelet"}
	case "peanut":
		return []string{"peanut", "groundnut"}
	case "tree_nut":
		return []string{"almond", "cashew", "walnut", "pistachio", "hazelnut"}
	case "soy":
		return []string{"soy", "tofu"}
	case "wheat":
		return []string{"wheat", "maida", "flour", "bread", "noodle", "pasta"}
	case "sesame":
		return []string{"sesame", "til"}
	case "fish":
		return []string{"fish", "fillet"}
	case "shellfish":
		return []string{"shellfish", "prawn", "shrimp", "crab", "lobster"}
	default:
		return []string{allergen}
	}
}

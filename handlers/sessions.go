package handlers

import (
	"encoding/json"
	"fitfuel/db"
	"fitfuel/models"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type UpdateSessionRequest struct {
	Age              int      `json:"age"`
	Gender           string   `json:"gender"`
	ActivityLevel    string   `json:"activity_level"`
	FitnessGoal      string   `json:"fitness_goal"`
	DietType         string   `json:"diet_type"`
	HeartRate        int      `json:"heart_rate"`
	Allergies        []string `json:"allergies"`
	RestrictionsText string   `json:"restrictions_text"`
	Restrictions     string   `json:"restrictions"`
}

func UpdateSession(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Session ID is required",
		})
	}

	var req UpdateSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	var session models.Session
	if err := db.DB.First(&session, "id = ?", id).Error; err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Session not found",
		})
	}

	// Update fields
	if req.Age > 0 {
		session.Age = req.Age
	}
	if req.Gender != "" {
		session.Gender = req.Gender
	}
	if req.ActivityLevel != "" {
		session.ActivityLevel = req.ActivityLevel
	}
	if req.FitnessGoal != "" {
		session.FitnessGoal = req.FitnessGoal
	}
	if req.DietType != "" {
		session.DietType = req.DietType
	}
	if req.HeartRate > 0 {
		session.HeartRate = req.HeartRate
	}

	allergyList := req.Allergies
	if len(allergyList) == 0 {
		allergyList = parseCSVList(req.RestrictionsText)
	}
	if len(allergyList) == 0 {
		allergyList = parseCSVList(req.Restrictions)
	}
	if len(allergyList) > 0 {
		rawAllergies, err := json.Marshal(allergyList)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid allergies payload",
			})
		}
		session.Allergies = rawAllergies
	}

	if req.RestrictionsText != "" {
		session.RestrictionsText = req.RestrictionsText
	} else if req.Restrictions != "" {
		session.RestrictionsText = req.Restrictions
	}

	if err := db.DB.Save(&session).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update session",
		})
	}

	return c.Status(http.StatusOK).JSON(session)
}

func parseCSVList(raw string) []string {
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	items := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))

	for _, part := range parts {
		item := strings.ToLower(strings.TrimSpace(part))
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		items = append(items, item)
	}

	return items
}

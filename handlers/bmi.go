package handlers

import (
	"fitfuel/db"
	"fitfuel/models"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

type BMIRequest struct {
	Weight float64 `json:"weight"`
	Height float64 `json:"height"`
}

type BMIResponse struct {
	SessionID string  `json:"sessionId"`
	BMI       float64 `json:"bmi"`
	Category  string  `json:"category"`
	Weight    float64 `json:"weight"`
	Height    float64 `json:"height"`
}

func CalculateBMI(c *fiber.Ctx) error {
	var req BMIRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Weight <= 0 || req.Height <= 0 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Weight and height must be positive numbers",
		})
	}

	// Calculate BMI
	bmi := req.Weight / ((req.Height / 100) * (req.Height / 100))

	// Determine category
	var category string
	if bmi < 18.5 {
		category = "Underweight"
	} else if bmi < 25 {
		category = "Normal weight"
	} else if bmi < 30 {
		category = "Overweight"
	} else {
		category = "Obese"
	}

	// Create session
	session := models.Session{
		BMI:    bmi,
		Weight: req.Weight,
		Height: req.Height,
	}

	if err := db.DB.Create(&session).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create session",
		})
	}

	return c.Status(http.StatusCreated).JSON(BMIResponse{
		SessionID: session.ID,
		BMI:       bmi,
		Category:  category,
		Weight:    req.Weight,
		Height:    req.Height,
	})
}

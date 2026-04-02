package handlers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

type Category struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Emoji string `json:"emoji"`
}

func GetCategories(c *fiber.Ctx) error {
	categories := []Category{
		{
			ID:    "veg",
			Label: "Vegetarian",
			Emoji: "🥦",
		},
		{
			ID:    "egg",
			Label: "Eggetarian",
			Emoji: "🍳",
		},
		{
			ID:    "nonveg",
			Label: "Non-Vegetarian",
			Emoji: "🍗",
		},
	}

	return c.Status(http.StatusOK).JSON(categories)
}

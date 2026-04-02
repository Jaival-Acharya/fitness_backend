package main

import (
	"fitfuel/db"
	"fitfuel/handlers"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Connect to database
	db.Connect()

	// Migrate database
	db.Migrate()

	// Seed database
	db.Seed()

	// Create Fiber app
	app := fiber.New()

	// Add middleware
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:5173,http://127.0.0.1:5173",
		AllowMethods: "GET,POST,PATCH,OPTIONS",
		AllowHeaders: "Content-Type",
	}))

	// API routes
	api := app.Group("/api")

	// BMI endpoint
	api.Post("/bmi", handlers.CalculateBMI)

	// Session endpoint
	api.Patch("/sessions/:id", handlers.UpdateSession)

	// Dishes endpoints
	api.Get("/dishes", handlers.GetDishes)
	api.Get("/dish/:id", handlers.GetDishByID)

	// Categories endpoint
	api.Get("/categories", handlers.GetCategories)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on port %s...", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

package main

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/Jake-Schuler/MoSim-Event-Manager/config"
	"github.com/Jake-Schuler/MoSim-Event-Manager/handlers"
	"github.com/Jake-Schuler/MoSim-Event-Manager/services"
)

//go:embed static/*
var static embed.FS

//go:embed templates/*
var templates embed.FS

func main() {
	// Load environment variables
	if err := godotenv.Load("data/.env"); err != nil {
		panic("Error loading .env file")
	}

	// Initialize database
	db := config.InitDB()

	// Initialize WebSocket state from database
	services.InitializeWebSocketState(db)

	// Initialize MMID counter based on existing users
	services.GetMMID(db)

	// Initialize Discord Bot
	dg := config.InitDiscordBot()

	// Initialize Gin router
	r := gin.Default()
	r.SetHTMLTemplate(template.Must(template.New("").ParseFS(templates, "templates/*")))
	r.StaticFS("/static", http.FS(static))

	// Setup routes
	handlers.SetupRoutes(r, db, dg)

	// Start server
	if err := r.Run(":8080"); err != nil {
		panic("failed to start server")
	}
}

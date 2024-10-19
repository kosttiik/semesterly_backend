package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/kosttiik/semesterly_backend/docs" // Swagger documentation
	"github.com/kosttiik/semesterly_backend/internal/pkg/app"

	"github.com/labstack/echo/v4"
)

// @title Автоматизированная система по ведению расписания учебных занятий
// @version 1.0
// @description API для управления расписанием учебных занятий
// @host localhost:8080
// @BasePath /api/v1
func main() {
	log.Println("Application started!")

	if err := godotenv.Load(); err != nil {
		log.Println("Failed to load .env file, trying to continue...")
	}

	// Создаем новое приложение
	a, err := app.New()
	if err != nil {
		log.Fatalf("Failed to initialize the app: %v", err)
	}

	e := echo.New()

	a.RegisterRoutes(e)

	// Запуск сервера
	go func() {
		if err := e.Start(":8080"); err != nil {
			log.Fatalf("Shutting down the server: %v", err)
		}
	}()

	handleShutdown(e)
}

func handleShutdown(e *echo.Echo) {
	// Ловим сигналы завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	if err := e.Shutdown(context.Background()); err != nil {
		log.Fatalf("Error shutting down server: %v", err)
	}

	time.Sleep(1 * time.Second)
}

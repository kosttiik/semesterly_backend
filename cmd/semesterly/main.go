package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	// Создаем новое приложение
	a, err := app.New()
	if err != nil {
		log.Fatalf("Failed to initialize the app: %v", err)
	}

	e := echo.New()
	a.RegisterRoutes(e)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Канал для передачи ошибок из горутины
	errChan := make(chan error, 1)

	// Запуск сервера в горутине
	go func() {
		if err := e.Start(":" + port); err != nil {
			errChan <- err
		}
	}()

	// Обработка завершения приложения
	handleShutdown(e, errChan)
}

func handleShutdown(e *echo.Echo, errChan chan error) {
	// Ловим сигналы завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Ожидание сигнала завершения или ошибки от сервера
	select {
	case err := <-errChan:
		log.Fatalf("Shutting down the server due to error: %v", err)
	case <-quit:
		log.Println("Shutting down server...")
		if err := e.Shutdown(context.Background()); err != nil {
			log.Fatalf("Error shutting down server: %v", err)
		}
	}

	// Чуть времени на завершение
	time.Sleep(1 * time.Second)
}

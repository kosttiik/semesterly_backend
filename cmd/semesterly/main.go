package main

import (
	"context"
	"log"
	"net/http"
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
		if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Ловим сигналы завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		log.Fatalf("Ошибка сервера: %v", err)
	case sig := <-quit:
		log.Printf("Получен сигнал завершения: %v. Завершаем работу сервера...", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := e.Shutdown(ctx); err != nil {
			log.Fatalf("Ошибка при завершении сервера: %v", err)
		}
		log.Println("Сервер успешно завершил работу.")
	}
}

package app

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	_ "github.com/kosttiik/semesterly_backend/docs" // Swagger documentation
	"github.com/kosttiik/semesterly_backend/internal/handlers"
	echoSwagger "github.com/swaggo/echo-swagger"
)

type App struct {
	DB *gorm.DB
}

var ErrMissingDatabaseConfig = errors.New("missing one or more required database environment variables")

// Инициализация приложения с подключением к БД
func New() (*App, error) {
	dbName := os.Getenv("DB_NAME")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")

	if dbName == "" || dbUser == "" || dbPassword == "" || dbHost == "" || dbPort == "" {
		return nil, ErrMissingDatabaseConfig
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable", dbHost, dbUser, dbPassword, dbName, dbPort)

	var db *gorm.DB
	var err error

	// Ожидание подключения к БД
	for i := 0; i < 10; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		waitTime := time.Duration(i+1) * time.Second
		log.Printf("Waiting for database... retrying in %v", waitTime)
		time.Sleep(waitTime)
	}

	if err != nil {
		return nil, err
	}

	log.Println("Connected to the database successfully!")

	return &App{
		DB: db,
	}, nil
}

func (a *App) RegisterRoutes(e *echo.Echo) {
	// Logger запросов в терминал
	e.Use(middleware.Logger())

	h := &handlers.App{
		DB: a.DB,
	}

	// Swagger documentation
	e.GET("/swagger*", echoSwagger.WrapHandler)

	e.GET("/api/v1/hello", h.HelloHandler)
}

package app

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	_ "github.com/kosttiik/semesterly_backend/docs" // Swagger documentation
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
	// Swagger documentation
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	e.GET("/api/v1/hello", a.helloHandler)
}

// helloHandler - обработчик для теста работы сервера
// @Summary Проверка подключения
// @Description Проверяет, работает ли сервер и есть ли подключение к базе данных
// @Tags Hello
// @Accept  json
// @Produce  json
// @Success 200 {string} string "Hello, World!"
// @Router /hello [get]
func (a *App) helloHandler(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World! Connected to the database successfully.")
}

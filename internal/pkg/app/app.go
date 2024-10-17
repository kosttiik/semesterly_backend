package app

import (
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	_ "github.com/kosttiik/semesterly_backend/docs" // Swagger
	echoSwagger "github.com/swaggo/echo-swagger"
)

type App struct {
	DB *gorm.DB
}

var ErrMissingDatabaseURL = errors.New("missing DATABASE_URL environment variable")

// New инициализирует приложение с подключением к БД
func New() (*App, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, ErrMissingDatabaseURL
	}

	var db *gorm.DB
	var err error

	// Ожидание подключения к БД
	for i := 0; i < 10; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}

		log.Println("Waiting for database to be ready for use...")
		time.Sleep(2 * time.Second)
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
	e.GET("/api/v1/hello", a.helloHandler)

	// Swagger
	e.GET("/swagger/*", echoSwagger.WrapHandler)
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

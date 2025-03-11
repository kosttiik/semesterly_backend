package app

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	_ "github.com/kosttiik/semesterly_backend/docs" // Swagger documentation
	"github.com/kosttiik/semesterly_backend/internal/handlers"
	"github.com/kosttiik/semesterly_backend/internal/models"
	echoSwagger "github.com/swaggo/echo-swagger"
)

type App struct {
	DB  *gorm.DB
	Hub *handlers.WebSocketHub
}

var (
	ErrMissingDatabaseConfig = errors.New("missing DATABASE_URL environment variable")
	ErrInvalidRetryConfig    = errors.New("invalid retry configuration")
)

// Инициализация приложения с подключением к БД
func New() (*App, error) {
	// Настраиваем формат логов глобально
	timeFormat := os.Getenv("LOG_TIME_FORMAT")
	if timeFormat == "" {
		timeFormat = "15:04:05 02.01.2006"
	}
	log.SetFlags(0) // Убираем стандартный префикс (дата/время)
	log.SetOutput(&customLogger{format: timeFormat})

	// Получаем DATABASE_URL из .env
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, ErrMissingDatabaseConfig
	}

	// Получаем настройки повторных попыток подключения
	maxRetriesStr := os.Getenv("DB_MAX_RETRIES")
	retryIntervalStr := os.Getenv("DB_RETRY_INTERVAL")

	// Значения по умолчанию
	maxRetries := 10
	retryInterval := 1 * time.Second

	if maxRetriesStr != "" {
		var err error
		maxRetries, err = strconv.Atoi(maxRetriesStr)
		if err != nil || maxRetries < 0 {
			return nil, fmt.Errorf("%w: DB_MAX_RETRIES must be a non-negative integer", ErrInvalidRetryConfig)
		}
	}

	if retryIntervalStr != "" {
		var err error
		retryIntervalSeconds, err := strconv.Atoi(retryIntervalStr)
		if err != nil || retryIntervalSeconds <= 0 {
			return nil, fmt.Errorf("%w: DB_RETRY_INTERVAL must be a positive integer in seconds", ErrInvalidRetryConfig)
		}
		retryInterval = time.Duration(retryIntervalSeconds) * time.Second
	}

	var db *gorm.DB
	var err error

	// Ожидание подключения к БД
	for i := range maxRetries {
		db, err = gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
		if err == nil {
			break
		}
		waitTime := retryInterval * time.Duration(i+1)
		log.Printf("Waiting for database... retrying in %v", waitTime)
		time.Sleep(waitTime)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database after %d retries: %w", maxRetries, err)
	}

	log.Println("Connected to the database successfully!")

	// Миграция БД
	err = db.AutoMigrate(&models.ScheduleItem{}, &models.Exam{})
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	hub := handlers.NewWebSocketHub()
	go hub.Run()

	return &App{
		DB:  db,
		Hub: hub,
	}, nil
}

func (a *App) RegisterRoutes(e *echo.Echo) {
	// Логирование запросов в терминал
	timeFormat := os.Getenv("LOG_TIME_FORMAT")
	if timeFormat == "" {
		timeFormat = "15:04:05 02.01.2006"
	}

	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "[${time_custom}] | ${status} | ${method} ${uri} | ${remote_ip} | ${latency_human}" +
			"\n   Error: ${error}\n",
		CustomTimeFormat: timeFormat,
		Output:           os.Stdout,
	}))

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE},
	}))

	h := &handlers.App{
		DB:  a.DB,
		Hub: a.Hub,
	}

	// Документация Swagger
	e.GET("/docs/*", echoSwagger.WrapHandler)

	e.GET("/api/v1/hello", h.HelloHandler)
	e.POST("/api/v1/insert-data", h.InsertDataHandler)
	e.POST("/api/v1/insert-group-schedule/:uuid", h.InsertGroupScheduleHandler)

	e.GET("/api/v1/get-groups", h.GetGroupsHandler)
	e.GET("/api/v1/get-data", h.GetDataHandler)
	e.GET("/api/v1/get-group-schedule/:uuid", h.GetGroupScheduleHandler)

	e.POST("/api/v1/write-schedule", h.WriteScheduleToFileHandler)

	e.GET("/ws", h.HandleWebSocket)
}

// customLogger для форматирования логов с использованием LOG_TIME_FORMAT
type customLogger struct {
	format string
}

func (cl *customLogger) Write(p []byte) (n int, err error) {
	currentTime := time.Now().Format(cl.format)
	_, err = fmt.Fprintf(os.Stdout, "[%s] %s", currentTime, string(p))
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

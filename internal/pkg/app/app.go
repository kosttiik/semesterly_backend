//go:build windows
// +build windows

package app

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"strconv"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
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
	// Значения по умолчанию для production
	const (
		defaultTimezone        = "Europe/Moscow"
		defaultLogTimeFormat   = "15:04:05 02.01.2006"
		defaultPort            = "8080"
		defaultDBName          = "postgres"
		defaultDBPort          = "5435"
		defaultDBUser          = "postgres"
		defaultDBPassword      = "postgres"
		defaultDBHost          = "localhost"
		defaultDBMaxRetries    = "10"
		defaultDBRetryInterval = "1"
	)

	// Устанавливаем переменные окружения, если не заданы
	if os.Getenv("TIMEZONE") == "" {
		os.Setenv("TIMEZONE", defaultTimezone)
	}
	if os.Getenv("LOG_TIME_FORMAT") == "" {
		os.Setenv("LOG_TIME_FORMAT", defaultLogTimeFormat)
	}
	if os.Getenv("PORT") == "" {
		os.Setenv("PORT", defaultPort)
	}
	if os.Getenv("DB_NAME") == "" {
		os.Setenv("DB_NAME", defaultDBName)
	}
	if os.Getenv("DB_PORT") == "" {
		os.Setenv("DB_PORT", defaultDBPort)
	}
	if os.Getenv("DB_USER") == "" {
		os.Setenv("DB_USER", defaultDBUser)
	}
	if os.Getenv("DB_PASSWORD") == "" {
		os.Setenv("DB_PASSWORD", defaultDBPassword)
	}
	if os.Getenv("DB_HOST") == "" {
		os.Setenv("DB_HOST", defaultDBHost)
	}
	if os.Getenv("DB_MAX_RETRIES") == "" {
		os.Setenv("DB_MAX_RETRIES", defaultDBMaxRetries)
	}
	if os.Getenv("DB_RETRY_INTERVAL") == "" {
		os.Setenv("DB_RETRY_INTERVAL", defaultDBRetryInterval)
	}

	// Настраиваем формат логов глобально
	timeFormat := os.Getenv("LOG_TIME_FORMAT")
	if timeFormat == "" {
		timeFormat = defaultLogTimeFormat
	}
	log.SetFlags(0)
	log.SetOutput(&customLogger{format: timeFormat})

	// Всегда запускаем embedded Postgre для production на 5435
	embeddedPg := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Username(defaultDBUser).
		Password(defaultDBPassword).
		Database(defaultDBName).
		Port(5435),
	)
	if startErr := embeddedPg.Start(); startErr != nil {
		return nil, fmt.Errorf("failed to start embedded postgre: %w", startErr)
	}
	log.Println("Embedded Postgre started on port 5435")

	// Формируем строку подключения к embedded Postgre
	databaseURL := fmt.Sprintf("postgres://%s:%s@localhost:%s/%s?sslmode=disable",
		defaultDBUser, defaultDBPassword, defaultDBPort, defaultDBName,
	)

	maxRetriesStr := os.Getenv("DB_MAX_RETRIES")
	retryIntervalStr := os.Getenv("DB_RETRY_INTERVAL")
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

	// Ожидание подключения к embedded Postgres
	for i := range maxRetries {
		db, err = gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
		if err == nil {
			break
		}
		waitTime := retryInterval * time.Duration(i+1)
		log.Printf("Waiting for embedded database... retrying in %v", waitTime)
		time.Sleep(waitTime)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to embedded database after %d retries: %w", maxRetries, err)
	}

	log.Println("Connected to the embedded database successfully!")

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

//go:embed build/dist/*
var embeddedFrontend embed.FS

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
		AllowOrigins: []string{"http://localhost:5173"},
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE, echo.OPTIONS},
		AllowHeaders: []string{
			"Content-Type", "Authorization", echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept,
			"x-cookie-jsessionid", "x-cookie-__portal3_login", "x-cookie-__portal3_info",
			"x-cookie-width", "x-cookie-tgc", "x-cookie-_csrf",
		},
		AllowCredentials: true,
	}))

	h := &handlers.App{
		DB:  a.DB,
		Hub: a.Hub,
	}

	// Документация Swagger
	e.GET("/docs/*", echoSwagger.WrapHandler)

	// Проверка соединения
	e.GET("/api/v1/ping", h.PingHandler)

	// Вставка всех записей расписания, отдельной группы
	e.POST("/api/v1/insert-data", h.InsertDataHandler)
	e.POST("/api/v1/insert-group-schedule/:uuid", h.InsertGroupScheduleHandler)

	// Удаление всех записей расписания
	e.POST("/api/v1/clear-data", h.ClearDataHandler)

	// Получение всех записей расписания
	e.GET("/api/v1/get-data", h.GetDataHandler)

	// Получение списка групп и расписания группы
	e.GET("/api/v1/get-groups", h.GetGroupsHandler)
	e.GET("/api/v1/get-group-schedule/:uuid", h.GetGroupScheduleHandler)

	// Получение списка преподавателей и расписания преподавателя
	e.GET("/api/v1/get-teachers", h.GetTeachersHandler)
	e.GET("/api/v1/get-teacher-schedule/:uuid", h.GetTeacherScheduleHandler)

	// Запись расписания в файл (CSV)
	e.POST("/api/v1/write-schedule", h.WriteScheduleToFileHandler)

	// WebSocket
	e.GET("/ws", h.HandleWebSocket)

	// Логин в LKS BMSTU
	e.POST("/api/v1/login-external", h.LoginExternalHandler)

	subFS, err := fs.Sub(embeddedFrontend, "build/dist")
	if err != nil {
		log.Fatalf("failed to prepare embedded frontend: %v", err)
	}

	// Корректная раздача статики с правильными Content-Type
	e.StaticFS("/assets", echo.MustSubFS(subFS, "assets"))
	e.FileFS("/favicon.ico", "favicon.ico", subFS)
	e.FileFS("/manifest.json", "manifest.json", subFS)

	e.GET("/", func(c echo.Context) error {
		f, err := subFS.Open("index.html")
		if err != nil {
			return echo.ErrNotFound
		}
		defer f.Close()
		return c.Stream(200, "text/html", f)
	})
	e.GET("/*", func(c echo.Context) error {
		f, err := subFS.Open("index.html")
		if err != nil {
			return echo.ErrNotFound
		}
		defer f.Close()
		return c.Stream(200, "text/html", f)
	})
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

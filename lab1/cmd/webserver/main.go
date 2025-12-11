package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"io"
	"lab1/internal/app/config"
	"lab1/internal/app/dsn"
	"lab1/internal/app/handler"
	"lab1/internal/app/redis"
	"lab1/internal/app/repository"
	"lab1/internal/pkg"
	"log"
	"net/http"
)

// @title ASPIRIN
// @version 1.0
// @description Aspirin

// @contact.name API Support
// @contact.url https://github.com/Qrp34ch/RIP
// @contact.email address

// @license.name AS IS (NO WARRANTY)
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Введите JWT токен в формате: Bearer {your_token}
// @host localhost:8080
// @schemes https
// @BasePath /

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Разрешенные адреса
		allowedOrigins := []string{
			"https://qrp34ch.github.io", // Ваш GitHub Pages
			"http://localhost:3000",     // Локальная разработка
			"http://localhost:5173",     // Vite dev server
			"https://localhost:3000",
			"http://192.168.0.102:3000", // Ваш IP
			"https://192.168.0.102:3000",
			"https://192.168.56.1:3000",
			"http://192.168.56.1:3000",
		}

		// Проверяем разрешен ли origin
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				//c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
				break
			}
		}

		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept, Origin")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func main() {
	router := gin.Default()

	// Используем CORS middleware вместо inline middleware
	router.Use(corsMiddleware())

	// Удалите старый inline CORS middleware (этот блок):
	// router.Use(func(c *gin.Context) {
	//     c.Header("Access-Control-Allow-Origin", "*")
	//     ...
	// })

	conf, err := config.NewConfig()
	if err != nil {
		logrus.Fatalf("error loading config: %v", err)
	}

	redisClient, err := redis.New(context.Background(), conf.Redis)
	if err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}
	defer redisClient.Close()

	postgresString := dsn.FromEnv()
	fmt.Println("PostgreSQL DSN:", postgresString)

	minioClient, err := conf.InitMinIO()
	if err != nil {
		logrus.Fatalf("error initializing MinIO: %v", err)
	}
	logrus.Info("MinIO client initialized successfully")

	rep, errRep := repository.New(postgresString, minioClient, conf.MinIOBucket, redisClient)
	if errRep != nil {
		logrus.Fatalf("error initializing repository: %v", errRep)
	}

	hand := handler.NewHandler(conf, rep)

	application := pkg.NewApp(conf, router, hand)

	// Прокси для MinIO с поддержкой CORS
	router.GET("/minio/*path", func(c *gin.Context) {
		path := c.Param("path")

		// Формируем URL до MinIO
		minioURL := fmt.Sprintf("http://localhost:9000%s", path)

		client := &http.Client{}

		req, err := http.NewRequest("GET", minioURL, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request to MinIO"})
			return
		}

		// Копируем заголовки запроса
		for key, values := range c.Request.Header {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}

		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to MinIO: " + err.Error()})
			return
		}
		defer resp.Body.Close()

		// Копируем заголовки ответа
		for key, values := range resp.Header {
			for _, value := range values {
				c.Header(key, value)
			}
		}

		// Добавляем CORS заголовки для изображений
		c.Header("Access-Control-Allow-Origin", c.Request.Header.Get("Origin"))
		c.Header("Access-Control-Allow-Credentials", "true")

		c.Status(resp.StatusCode)

		_, err = io.Copy(c.Writer, resp.Body)
		if err != nil {
			logrus.Errorf("Failed to copy MinIO response: %v", err)
		}

	})

	application.RunApp()
}

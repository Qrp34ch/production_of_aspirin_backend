package main

import (
	"context"
	"fmt"
	"lab1/internal/app/redis"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"lab1/internal/app/config"
	"lab1/internal/app/dsn"
	"lab1/internal/app/handler"
	"lab1/internal/app/repository"
	"lab1/internal/pkg"
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
// @schemes https http
// @BasePath /

func main() {
	router := gin.Default()
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, Accept")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})
	conf, err := config.NewConfig()
	if err != nil {
		logrus.Fatalf("error loading config: %v", err)
	}
	redisClient, err := redis.New(context.Background(), conf.Redis)

	//redisClient, err := redis.New(context.Background(), *conf)
	if err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}
	defer redisClient.Close()

	postgresString := dsn.FromEnv()
	fmt.Println(postgresString)

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
	application.RunApp()
}

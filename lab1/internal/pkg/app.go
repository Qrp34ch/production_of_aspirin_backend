package pkg

import (
	"fmt"
	"os"

	"lab1/internal/app/config"
	"lab1/internal/app/handler"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Application struct {
	Config  *config.Config
	Router  *gin.Engine
	Handler *handler.Handler
}

func NewApp(c *config.Config, r *gin.Engine, h *handler.Handler) *Application {
	return &Application{
		Config:  c,
		Router:  r,
		Handler: h,
	}
}

func (a *Application) RunApp() {
	logrus.Info("Server starting up...")

	a.Handler.RegisterHandler(a.Router)

	// Используем порт 8443 для HTTPS (стандартный порт для HTTPS при разработке)
	// Порт 443 требует прав администратора
	if a.Config.ServicePort == 8080 {
		//a.Config.ServicePort = 8443
		a.Config.ServicePort = 8080
	}

	serverAddress := fmt.Sprintf("%s:%d", a.Config.ServiceHost, a.Config.ServicePort)

	// Проверяем наличие SSL сертификатов
	certFile := "cert.crt"
	keyFile := "cert.key"

	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		logrus.Fatalf("SSL certificate '%s' not found. Generate it first.", certFile)
	}
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		logrus.Fatalf("SSL key '%s' not found. Generate it first.", keyFile)
	}

	logrus.Infof("Starting HTTPS server on https://%s", serverAddress)

	// Запускаем HTTPS сервер
	if err := a.Router.RunTLS(serverAddress, certFile, keyFile); err != nil {
		logrus.Fatal("HTTPS server failed to start: ", err)
	}

	logrus.Info("Server shut down")
}

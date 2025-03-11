package main

import (
	"flag"
	"github.com/Krchnk/gw-currency-wallet/internal/config"
	"github.com/Krchnk/gw-currency-wallet/internal/handlers"
	"github.com/Krchnk/gw-currency-wallet/internal/storages/postgres"
	"github.com/gin-gonic/gin"
	pb "github.com/proto-exchange/exchange_grpc"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"os"
	"time"
)

var logger = logrus.New()

func init() {
	logger.SetFormatter(&logrus.JSONFormatter{})
	if lvl, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL")); err == nil {
		logger.SetLevel(lvl)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}
}

func main() {
	configPath := flag.String("c", "config.env", "path to config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logger.WithError(err).Fatal("failed to load config")
	}
	logger.WithField("config", cfg).Info("configuration loaded")

	store, err := postgres.NewStorage(cfg.DBConfig)
	if err != nil {
		logger.WithError(err).Fatal("failed to connect to database")
	}
	logger.Info("database connection established")

	conn, err := grpc.Dial(cfg.ExchangeServiceAddr, grpc.WithInsecure())
	if err != nil {
		logger.WithError(err).Fatal("failed to connect to exchange service")
	}
	defer conn.Close()
	exchangeClient := pb.NewExchangeServiceClient(conn)

	router := gin.Default()
	router.Use(loggingMiddleware())

	h := handlers.NewHandler(store, exchangeClient, cfg)

	api := router.Group("/api/v1")
	{
		api.POST("/register", h.Register)
		api.POST("/login", h.Login)

		auth := api.Group("", h.AuthMiddleware())
		{
			auth.GET("/balance", h.GetBalance)
			auth.POST("/wallet/deposit", h.Deposit)
			auth.POST("/wallet/withdraw", h.Withdraw)
			auth.GET("/exchange/rates", h.GetRates)
			auth.POST("/exchange", h.Exchange)
		}
	}

	logger.WithField("port", cfg.HTTPPort).Info("starting HTTP server")
	if err := router.Run(cfg.HTTPPort); err != nil {
		logger.WithError(err).Fatal("failed to run server")
	}
}

func loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		logger.WithFields(logrus.Fields{
			"method": c.Request.Method,
			"path":   path,
		}).Info("request received")

		c.Next()

		duration := time.Since(start)
		fields := logrus.Fields{
			"method":   c.Request.Method,
			"path":     path,
			"status":   c.Writer.Status(),
			"duration": duration,
		}

		if len(c.Errors) > 0 {
			logger.WithFields(fields).WithError(c.Errors.Last()).Error("request failed")
		} else {
			logger.WithFields(fields).Info("request completed")
		}
	}
}

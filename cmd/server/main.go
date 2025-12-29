package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sebaespinosa/test_NF_auto/internal/controller"
	"github.com/sebaespinosa/test_NF_auto/internal/model"
	"github.com/sebaespinosa/test_NF_auto/internal/repository"
	"github.com/sebaespinosa/test_NF_auto/internal/service"
	"github.com/sebaespinosa/test_NF_auto/pkg/logger"
	"github.com/sebaespinosa/test_NF_auto/pkg/metrics"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Initialize logger
	appLogger := logger.New()
	appLogger.Info("Starting Irrigation Analytics Service", map[string]interface{}{})

	// Initialize metrics
	appMetrics := metrics.New()

	// Get database connection string from environment
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "irrigation_analytics")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	// Connect to database
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto migrate database schema
	err = db.AutoMigrate(&model.Farm{}, &model.IrrigationSector{}, &model.IrrigationData{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	appLogger.Info("Database migration completed", map[string]interface{}{})

	// Initialize repository
	irrigationRepo := repository.NewIrrigationRepository(db)

	// Initialize service
	analyticsService := service.NewAnalyticsService(irrigationRepo)

	// Initialize controller
	irrigationController := controller.NewIrrigationController(analyticsService, appLogger, appMetrics)

	// Setup Gin router
	router := gin.Default()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	// Metrics endpoint
	router.GET("/metrics", func(c *gin.Context) {
		c.JSON(200, appMetrics.GetStats())
	})

	// API v1 routes
	v1 := router.Group("/v1")
	{
		farms := v1.Group("/farms")
		{
			farms.GET("/:farm_id/irrigation/analytics", irrigationController.GetAnalytics)
		}
	}

	// Get server configuration
	port := getEnv("PORT", "8443")
	certFile := getEnv("TLS_CERT", "")
	keyFile := getEnv("TLS_KEY", "")

	appLogger.Info("Starting server", map[string]interface{}{
		"port": port,
		"tls":  certFile != "" && keyFile != "",
	})

	// Start server (HTTPS if cert/key provided, otherwise HTTP)
	if certFile != "" && keyFile != "" {
		if err := router.RunTLS(":"+port, certFile, keyFile); err != nil {
			log.Fatalf("Failed to start HTTPS server: %v", err)
		}
	} else {
		// For development, generate self-signed certificates or run HTTP
		appLogger.Warn("Running without TLS", map[string]interface{}{
			"message": "Set TLS_CERT and TLS_KEY environment variables for HTTPS",
		})
		if err := router.Run(":" + port); err != nil {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

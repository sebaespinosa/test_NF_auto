package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sebaespinosa/test_NF_auto/internal/service"
	"github.com/sebaespinosa/test_NF_auto/pkg/logger"
	"github.com/sebaespinosa/test_NF_auto/pkg/metrics"
)

// IrrigationController handles irrigation-related HTTP requests
type IrrigationController struct {
	service service.AnalyticsService
	logger  *logger.Logger
	metrics *metrics.Metrics
}

// NewIrrigationController creates a new irrigation controller
func NewIrrigationController(service service.AnalyticsService, log *logger.Logger, met *metrics.Metrics) *IrrigationController {
	return &IrrigationController{
		service: service,
		logger:  log,
		metrics: met,
	}
}

// GetAnalytics handles GET /v1/farms/:farm_id/irrigation/analytics
func (c *IrrigationController) GetAnalytics(ctx *gin.Context) {
	startTime := time.Now()

	// Parse farm_id
	farmIDParam := ctx.Param("farm_id")
	farmID, err := strconv.ParseUint(farmIDParam, 10, 32)
	if err != nil {
		c.logger.Error("Invalid farm_id", map[string]interface{}{
			"farm_id": farmIDParam,
			"error":   err.Error(),
		})
		c.metrics.RecordError("/v1/farms/:farm_id/irrigation/analytics")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid farm_id"})
		return
	}

	// Parse query parameters
	startDateStr := ctx.Query("start_date")
	endDateStr := ctx.Query("end_date")
	sectorIDStr := ctx.Query("sector_id")
	aggregation := ctx.DefaultQuery("aggregation", "daily")

	// Parse dates
	var startDate, endDate time.Time
	if startDateStr != "" {
		startDate, err = time.Parse(time.RFC3339, startDateStr)
		if err != nil {
			// Try parsing as date only
			startDate, err = time.Parse("2006-01-02", startDateStr)
			if err != nil {
				c.logger.Error("Invalid start_date", map[string]interface{}{
					"start_date": startDateStr,
					"error":      err.Error(),
				})
				c.metrics.RecordError("/v1/farms/:farm_id/irrigation/analytics")
				ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date format. Use ISO 8601 (e.g., 2024-01-01 or 2024-01-01T00:00:00Z)"})
				return
			}
		}
	} else {
		// Default to 30 days ago
		startDate = time.Now().AddDate(0, 0, -30)
	}

	if endDateStr != "" {
		endDate, err = time.Parse(time.RFC3339, endDateStr)
		if err != nil {
			// Try parsing as date only
			endDate, err = time.Parse("2006-01-02", endDateStr)
			if err != nil {
				c.logger.Error("Invalid end_date", map[string]interface{}{
					"end_date": endDateStr,
					"error":    err.Error(),
				})
				c.metrics.RecordError("/v1/farms/:farm_id/irrigation/analytics")
				ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date format. Use ISO 8601 (e.g., 2024-01-31 or 2024-01-31T23:59:59Z)"})
				return
			}
		}
	} else {
		// Default to now
		endDate = time.Now()
	}

	// Validate date range
	if endDate.Before(startDate) {
		c.logger.Error("Invalid date range", map[string]interface{}{
			"start_date": startDate,
			"end_date":   endDate,
		})
		c.metrics.RecordError("/v1/farms/:farm_id/irrigation/analytics")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "end_date must be after start_date"})
		return
	}

	// Parse sector_id if provided
	var sectorID *uint
	if sectorIDStr != "" {
		sectorIDVal, err := strconv.ParseUint(sectorIDStr, 10, 32)
		if err != nil {
			c.logger.Error("Invalid sector_id", map[string]interface{}{
				"sector_id": sectorIDStr,
				"error":     err.Error(),
			})
			c.metrics.RecordError("/v1/farms/:farm_id/irrigation/analytics")
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sector_id"})
			return
		}
		sectorIDUint := uint(sectorIDVal)
		sectorID = &sectorIDUint
	}

	// Validate aggregation
	if aggregation != "daily" && aggregation != "weekly" && aggregation != "monthly" {
		c.logger.Error("Invalid aggregation", map[string]interface{}{
			"aggregation": aggregation,
		})
		c.metrics.RecordError("/v1/farms/:farm_id/irrigation/analytics")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid aggregation. Must be one of: daily, weekly, monthly"})
		return
	}

	// Log request
	c.logger.Info("Processing analytics request", map[string]interface{}{
		"farm_id":     farmID,
		"start_date":  startDate.Format(time.RFC3339),
		"end_date":    endDate.Format(time.RFC3339),
		"aggregation": aggregation,
	})

	// Get analytics
	result, err := c.service.GetIrrigationAnalytics(uint(farmID), startDate, endDate, sectorID, aggregation)
	if err != nil {
		c.logger.Error("Failed to get analytics", map[string]interface{}{
			"farm_id": farmID,
			"error":   err.Error(),
		})
		c.metrics.RecordError("/v1/farms/:farm_id/irrigation/analytics")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve analytics"})
		return
	}

	// Record metrics
	latency := time.Since(startTime)
	c.metrics.RecordRequest("/v1/farms/:farm_id/irrigation/analytics", latency)

	// Log success
	c.logger.Info("Analytics request completed", map[string]interface{}{
		"farm_id":      farmID,
		"latency_ms":   latency.Milliseconds(),
		"result_count": len(result.TimeSeries),
	})

	ctx.JSON(http.StatusOK, result)
}

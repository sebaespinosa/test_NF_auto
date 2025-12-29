package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sebaespinosa/test_NF_auto/internal/repository"
	"github.com/sebaespinosa/test_NF_auto/internal/service"
	"github.com/sebaespinosa/test_NF_auto/pkg/logger"
	"github.com/sebaespinosa/test_NF_auto/pkg/metrics"
)

// mockIrrigationRepository for integration testing
type mockIrrigationRepository struct{}

func (m *mockIrrigationRepository) GetAnalytics(farmID uint, startDate, endDate time.Time, sectorID *uint) (*repository.AnalyticsData, error) {
	// Return sample data based on year
	year := startDate.Year()
	
	baseVolume := 450.5
	baseEvents := int64(120)
	baseEfficiency := 0.85
	
	if year == 2023 {
		baseVolume = 420.3
		baseEvents = 115
		baseEfficiency = 0.82
	} else if year == 2022 {
		baseVolume = 480.1
		baseEvents = 125
		baseEfficiency = 0.88
	}
	
	return &repository.AnalyticsData{
		TotalVolume:       baseVolume,
		TotalEvents:       baseEvents,
		AverageEfficiency: baseEfficiency,
		MinEfficiency:     0.72,
		MaxEfficiency:     0.98,
	}, nil
}

func (m *mockIrrigationRepository) GetTimeSeriesData(farmID uint, startDate, endDate time.Time, sectorID *uint, aggregation string) ([]repository.TimeSeriesEntry, error) {
	return []repository.TimeSeriesEntry{
		{Date: "2024-01-01", NominalAmount: 12.5, RealAmount: 10.8, Efficiency: 0.864, EventCount: 3},
		{Date: "2024-01-02", NominalAmount: 15.0, RealAmount: 13.2, Efficiency: 0.880, EventCount: 4},
	}, nil
}

func (m *mockIrrigationRepository) GetSectorBreakdown(farmID uint, startDate, endDate time.Time) ([]repository.SectorBreakdown, error) {
	return []repository.SectorBreakdown{
		{SectorID: 1, SectorName: "Sector A", TotalVolume: 150.2, AverageEfficiency: 0.88},
		{SectorID: 2, SectorName: "Sector B", TotalVolume: 200.3, AverageEfficiency: 0.85},
	}, nil
}

func TestGetAnalytics_Integration(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	
	mockRepo := &mockIrrigationRepository{}
	analyticsService := service.NewAnalyticsService(mockRepo)
	controller := NewIrrigationController(analyticsService, logger.New(), metrics.New())
	
	router := gin.New()
	router.GET("/v1/farms/:farm_id/irrigation/analytics", controller.GetAnalytics)
	
	// Test successful request
	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/v1/farms/1/irrigation/analytics?start_date=2024-01-01&end_date=2024-01-31&aggregation=daily", nil)
		w := httptest.NewRecorder()
		
		router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		
		if response["farm_id"].(float64) != 1 {
			t.Errorf("Expected farm_id 1, got %v", response["farm_id"])
		}
		
		metrics := response["metrics"].(map[string]interface{})
		if metrics["total_irrigation_volume_mm"].(float64) != 450.5 {
			t.Errorf("Expected volume 450.5, got %v", metrics["total_irrigation_volume_mm"])
		}
	})
	
	// Test invalid farm ID
	t.Run("InvalidFarmID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/v1/farms/invalid/irrigation/analytics", nil)
		w := httptest.NewRecorder()
		
		router.ServeHTTP(w, req)
		
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
	
	// Test invalid date format
	t.Run("InvalidDateFormat", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/v1/farms/1/irrigation/analytics?start_date=invalid", nil)
		w := httptest.NewRecorder()
		
		router.ServeHTTP(w, req)
		
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
	
	// Test invalid aggregation
	t.Run("InvalidAggregation", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/v1/farms/1/irrigation/analytics?aggregation=invalid", nil)
		w := httptest.NewRecorder()
		
		router.ServeHTTP(w, req)
		
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
	
	// Test with sector filter
	t.Run("WithSectorFilter", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/v1/farms/1/irrigation/analytics?sector_id=1&start_date=2024-01-01&end_date=2024-01-31", nil)
		w := httptest.NewRecorder()
		
		router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

package service

import (
	"testing"
	"time"

	"github.com/sebaespinosa/test_NF_auto/internal/repository"
)

// mockIrrigationRepository is a mock implementation of IrrigationRepository for testing
type mockIrrigationRepository struct {
	analyticsData   map[string]*repository.AnalyticsData
	timeSeriesData  []repository.TimeSeriesEntry
	sectorBreakdown []repository.SectorBreakdown
}

func (m *mockIrrigationRepository) GetAnalytics(farmID uint, startDate, endDate time.Time, sectorID *uint) (*repository.AnalyticsData, error) {
	// Generate a key based on the start date to differentiate periods
	key := startDate.Format("2006-01-02")
	if data, exists := m.analyticsData[key]; exists {
		return data, nil
	}
	// Return empty data if not found
	return &repository.AnalyticsData{
		TotalVolume:       0,
		TotalEvents:       0,
		AverageEfficiency: 0,
		MinEfficiency:     0,
		MaxEfficiency:     0,
	}, nil
}

func (m *mockIrrigationRepository) GetTimeSeriesData(farmID uint, startDate, endDate time.Time, sectorID *uint, aggregation string) ([]repository.TimeSeriesEntry, error) {
	return m.timeSeriesData, nil
}

func (m *mockIrrigationRepository) GetSectorBreakdown(farmID uint, startDate, endDate time.Time) ([]repository.SectorBreakdown, error) {
	return m.sectorBreakdown, nil
}

func TestGetIrrigationAnalytics_Success(t *testing.T) {
	// Setup mock repository
	mockRepo := &mockIrrigationRepository{
		analyticsData: map[string]*repository.AnalyticsData{
			"2024-01-01": {
				TotalVolume:       450.5,
				TotalEvents:       120,
				AverageEfficiency: 0.85,
				MinEfficiency:     0.72,
				MaxEfficiency:     0.98,
			},
			"2023-01-01": {
				TotalVolume:       420.3,
				TotalEvents:       115,
				AverageEfficiency: 0.82,
				MinEfficiency:     0.70,
				MaxEfficiency:     0.95,
			},
			"2022-01-01": {
				TotalVolume:       480.1,
				TotalEvents:       125,
				AverageEfficiency: 0.88,
				MinEfficiency:     0.75,
				MaxEfficiency:     0.99,
			},
		},
		timeSeriesData: []repository.TimeSeriesEntry{
			{Date: "2024-01-01", NominalAmount: 12.5, RealAmount: 10.8, Efficiency: 0.864, EventCount: 3},
		},
		sectorBreakdown: []repository.SectorBreakdown{
			{SectorID: 1, SectorName: "Sector A", TotalVolume: 150.2, AverageEfficiency: 0.88},
		},
	}

	service := NewAnalyticsService(mockRepo)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	result, err := service.GetIrrigationAnalytics(1, startDate, endDate, nil, "daily")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.FarmID != 1 {
		t.Errorf("Expected farm_id 1, got %d", result.FarmID)
	}

	if result.Metrics.TotalIrrigationVolumeMM != 450.5 {
		t.Errorf("Expected total volume 450.5, got %f", result.Metrics.TotalIrrigationVolumeMM)
	}

	if result.Metrics.TotalIrrigationEvents != 120 {
		t.Errorf("Expected total events 120, got %d", result.Metrics.TotalIrrigationEvents)
	}

	if result.Metrics.AverageEfficiency != 0.85 {
		t.Errorf("Expected average efficiency 0.85, got %f", result.Metrics.AverageEfficiency)
	}

	// Check year-over-year comparisons
	if result.Metrics.SamePeriodMinus1 == nil {
		t.Error("Expected same_period_-1 to be populated")
	} else {
		if result.Metrics.SamePeriodMinus1.TotalIrrigationVolumeMM != 420.3 {
			t.Errorf("Expected period -1 volume 420.3, got %f", result.Metrics.SamePeriodMinus1.TotalIrrigationVolumeMM)
		}
	}

	if result.Metrics.SamePeriodMinus2 == nil {
		t.Error("Expected same_period_-2 to be populated")
	}

	// Check period comparisons
	if result.Metrics.PeriodComparison == nil {
		t.Error("Expected period_comparison to be populated")
	} else {
		if result.Metrics.PeriodComparison.VsSamePeriodMinus1 == nil {
			t.Error("Expected vs_same_period_-1 comparison to be populated")
		}
	}
}

func TestCalculatePercentChange(t *testing.T) {
	tests := []struct {
		name     string
		current  float64
		previous float64
		expected float64
	}{
		{"Increase", 100, 80, 25.0},
		{"Decrease", 80, 100, -20.0},
		{"No change", 100, 100, 0.0},
		{"Zero previous", 100, 0, 0.0},
		{"Both zero", 0, 0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculatePercentChange(tt.current, tt.previous)
			if result != tt.expected {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestCalculatePreviousYearDates(t *testing.T) {
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	// Test -1 year
	prevStart1, prevEnd1 := calculatePreviousYearDates(startDate, endDate, 1)
	expectedStart1 := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	expectedEnd1 := time.Date(2023, 1, 31, 23, 59, 59, 0, time.UTC)

	if !prevStart1.Equal(expectedStart1) {
		t.Errorf("Expected start date %v, got %v", expectedStart1, prevStart1)
	}
	if !prevEnd1.Equal(expectedEnd1) {
		t.Errorf("Expected end date %v, got %v", expectedEnd1, prevEnd1)
	}

	// Test -2 years
	prevStart2, prevEnd2 := calculatePreviousYearDates(startDate, endDate, 2)
	expectedStart2 := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	expectedEnd2 := time.Date(2022, 1, 31, 23, 59, 59, 0, time.UTC)

	if !prevStart2.Equal(expectedStart2) {
		t.Errorf("Expected start date %v, got %v", expectedStart2, prevStart2)
	}
	if !prevEnd2.Equal(expectedEnd2) {
		t.Errorf("Expected end date %v, got %v", expectedEnd2, prevEnd2)
	}
}

func TestGetIrrigationAnalytics_InvalidAggregation(t *testing.T) {
	mockRepo := &mockIrrigationRepository{
		analyticsData: make(map[string]*repository.AnalyticsData),
	}

	service := NewAnalyticsService(mockRepo)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	_, err := service.GetIrrigationAnalytics(1, startDate, endDate, nil, "invalid")

	if err == nil {
		t.Error("Expected error for invalid aggregation, got nil")
	}
}

func TestGetIrrigationAnalytics_NoHistoricalData(t *testing.T) {
	// Setup mock repository with only current period data
	mockRepo := &mockIrrigationRepository{
		analyticsData: map[string]*repository.AnalyticsData{
			"2024-01-01": {
				TotalVolume:       450.5,
				TotalEvents:       120,
				AverageEfficiency: 0.85,
				MinEfficiency:     0.72,
				MaxEfficiency:     0.98,
			},
		},
		timeSeriesData:  []repository.TimeSeriesEntry{},
		sectorBreakdown: []repository.SectorBreakdown{},
	}

	service := NewAnalyticsService(mockRepo)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	result, err := service.GetIrrigationAnalytics(1, startDate, endDate, nil, "daily")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// When there's no historical data (TotalEvents = 0), the fields should be nil
	if result.Metrics.SamePeriodMinus1 != nil {
		t.Error("Expected same_period_-1 to be nil when no data exists")
	}

	if result.Metrics.SamePeriodMinus2 != nil {
		t.Error("Expected same_period_-2 to be nil when no data exists")
	}

	if result.Metrics.PeriodComparison != nil {
		t.Error("Expected period_comparison to be nil when no historical data exists")
	}
}

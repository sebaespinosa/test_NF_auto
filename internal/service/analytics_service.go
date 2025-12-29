package service

import (
	"fmt"
	"time"

	"github.com/sebaespinosa/test_NF_auto/internal/repository"
)

// AnalyticsService defines the interface for analytics business logic
type AnalyticsService interface {
	GetIrrigationAnalytics(farmID uint, startDate, endDate time.Time, sectorID *uint, aggregation string) (*IrrigationAnalyticsResponse, error)
}

// IrrigationAnalyticsResponse represents the complete analytics response
type IrrigationAnalyticsResponse struct {
	FarmID          uint                   `json:"farm_id"`
	Period          Period                 `json:"period"`
	Aggregation     string                 `json:"aggregation"`
	Metrics         Metrics                `json:"metrics"`
	TimeSeries      []TimeSeriesDataPoint  `json:"time_series"`
	SectorBreakdown []SectorBreakdownData  `json:"sector_breakdown"`
}

// Period represents the date range
type Period struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// Metrics represents all metrics including current and historical periods
type Metrics struct {
	TotalIrrigationVolumeMM float64           `json:"total_irrigation_volume_mm"`
	TotalIrrigationEvents   int64             `json:"total_irrigation_events"`
	AverageEfficiency       float64           `json:"average_efficiency"`
	EfficiencyRange         EfficiencyRange   `json:"efficiency_range"`
	SamePeriodMinus1        *PeriodMetrics    `json:"same_period_-1,omitempty"`
	SamePeriodMinus2        *PeriodMetrics    `json:"same_period_-2,omitempty"`
	PeriodComparison        *PeriodComparison `json:"period_comparison,omitempty"`
}

// EfficiencyRange represents min and max efficiency
type EfficiencyRange struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// PeriodMetrics represents metrics for a historical period
type PeriodMetrics struct {
	TotalIrrigationVolumeMM float64         `json:"total_irrigation_volume_mm"`
	TotalIrrigationEvents   int64           `json:"total_irrigation_events"`
	AverageEfficiency       float64         `json:"average_efficiency"`
	EfficiencyRange         EfficiencyRange `json:"efficiency_range"`
}

// PeriodComparison represents year-over-year comparisons
type PeriodComparison struct {
	VsSamePeriodMinus1 *ComparisonMetrics `json:"vs_same_period_-1,omitempty"`
	VsSamePeriodMinus2 *ComparisonMetrics `json:"vs_same_period_-2,omitempty"`
}

// ComparisonMetrics represents percentage changes
type ComparisonMetrics struct {
	VolumeChangePercent     float64 `json:"volume_change_percent"`
	EventsChangePercent     float64 `json:"events_change_percent"`
	EfficiencyChangePercent float64 `json:"efficiency_change_percent"`
}

// TimeSeriesDataPoint represents a single time-series entry
type TimeSeriesDataPoint struct {
	Date          string  `json:"date"`
	NominalAmount float64 `json:"nominal_amount_mm"`
	RealAmount    float64 `json:"real_amount_mm"`
	Efficiency    float64 `json:"efficiency"`
	EventCount    int64   `json:"event_count"`
}

// SectorBreakdownData represents sector-level aggregated data
type SectorBreakdownData struct {
	SectorID          uint    `json:"sector_id"`
	SectorName        string  `json:"sector_name"`
	TotalVolume       float64 `json:"total_volume_mm"`
	AverageEfficiency float64 `json:"average_efficiency"`
}

type analyticsService struct {
	repo repository.IrrigationRepository
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(repo repository.IrrigationRepository) AnalyticsService {
	return &analyticsService{repo: repo}
}

// GetIrrigationAnalytics retrieves comprehensive irrigation analytics
func (s *analyticsService) GetIrrigationAnalytics(farmID uint, startDate, endDate time.Time, sectorID *uint, aggregation string) (*IrrigationAnalyticsResponse, error) {
	// Validate aggregation
	if aggregation == "" {
		aggregation = "daily"
	}
	if aggregation != "daily" && aggregation != "weekly" && aggregation != "monthly" {
		return nil, fmt.Errorf("invalid aggregation level: %s", aggregation)
	}

	// Get current period metrics
	currentMetrics, err := s.repo.GetAnalytics(farmID, startDate, endDate, sectorID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current metrics: %w", err)
	}

	// Get time series data
	timeSeries, err := s.repo.GetTimeSeriesData(farmID, startDate, endDate, sectorID, aggregation)
	if err != nil {
		return nil, fmt.Errorf("failed to get time series data: %w", err)
	}

	// Get sector breakdown (only if not filtering by specific sector)
	var sectorBreakdown []repository.SectorBreakdown
	if sectorID == nil {
		sectorBreakdown, err = s.repo.GetSectorBreakdown(farmID, startDate, endDate)
		if err != nil {
			return nil, fmt.Errorf("failed to get sector breakdown: %w", err)
		}
	}

	// Calculate year-over-year metrics
	period1Start, period1End := calculatePreviousYearDates(startDate, endDate, 1)
	period2Start, period2End := calculatePreviousYearDates(startDate, endDate, 2)

	period1Metrics, err := s.repo.GetAnalytics(farmID, period1Start, period1End, sectorID)
	if err != nil {
		return nil, fmt.Errorf("failed to get period -1 metrics: %w", err)
	}

	period2Metrics, err := s.repo.GetAnalytics(farmID, period2Start, period2End, sectorID)
	if err != nil {
		return nil, fmt.Errorf("failed to get period -2 metrics: %w", err)
	}

	// Build response
	response := &IrrigationAnalyticsResponse{
		FarmID: farmID,
		Period: Period{
			Start: startDate.Format(time.RFC3339),
			End:   endDate.Format(time.RFC3339),
		},
		Aggregation: aggregation,
		Metrics: Metrics{
			TotalIrrigationVolumeMM: currentMetrics.TotalVolume,
			TotalIrrigationEvents:   currentMetrics.TotalEvents,
			AverageEfficiency:       currentMetrics.AverageEfficiency,
			EfficiencyRange: EfficiencyRange{
				Min: currentMetrics.MinEfficiency,
				Max: currentMetrics.MaxEfficiency,
			},
		},
		TimeSeries:      convertTimeSeries(timeSeries),
		SectorBreakdown: convertSectorBreakdown(sectorBreakdown),
	}

	// Add historical periods if data exists
	if period1Metrics.TotalEvents > 0 {
		response.Metrics.SamePeriodMinus1 = &PeriodMetrics{
			TotalIrrigationVolumeMM: period1Metrics.TotalVolume,
			TotalIrrigationEvents:   period1Metrics.TotalEvents,
			AverageEfficiency:       period1Metrics.AverageEfficiency,
			EfficiencyRange: EfficiencyRange{
				Min: period1Metrics.MinEfficiency,
				Max: period1Metrics.MaxEfficiency,
			},
		}
	}

	if period2Metrics.TotalEvents > 0 {
		response.Metrics.SamePeriodMinus2 = &PeriodMetrics{
			TotalIrrigationVolumeMM: period2Metrics.TotalVolume,
			TotalIrrigationEvents:   period2Metrics.TotalEvents,
			AverageEfficiency:       period2Metrics.AverageEfficiency,
			EfficiencyRange: EfficiencyRange{
				Min: period2Metrics.MinEfficiency,
				Max: period2Metrics.MaxEfficiency,
			},
		}
	}

	// Calculate period comparisons
	response.Metrics.PeriodComparison = calculatePeriodComparison(currentMetrics, period1Metrics, period2Metrics)

	return response, nil
}

// calculatePreviousYearDates calculates the date range for N years ago
func calculatePreviousYearDates(startDate, endDate time.Time, yearsAgo int) (time.Time, time.Time) {
	return startDate.AddDate(-yearsAgo, 0, 0), endDate.AddDate(-yearsAgo, 0, 0)
}

// calculatePeriodComparison calculates percentage changes between periods
func calculatePeriodComparison(current, period1, period2 *repository.AnalyticsData) *PeriodComparison {
	comparison := &PeriodComparison{}

	// Compare with period -1
	if period1.TotalEvents > 0 {
		comparison.VsSamePeriodMinus1 = &ComparisonMetrics{
			VolumeChangePercent:     calculatePercentChange(current.TotalVolume, period1.TotalVolume),
			EventsChangePercent:     calculatePercentChange(float64(current.TotalEvents), float64(period1.TotalEvents)),
			EfficiencyChangePercent: calculatePercentChange(current.AverageEfficiency, period1.AverageEfficiency),
		}
	}

	// Compare with period -2
	if period2.TotalEvents > 0 {
		comparison.VsSamePeriodMinus2 = &ComparisonMetrics{
			VolumeChangePercent:     calculatePercentChange(current.TotalVolume, period2.TotalVolume),
			EventsChangePercent:     calculatePercentChange(float64(current.TotalEvents), float64(period2.TotalEvents)),
			EfficiencyChangePercent: calculatePercentChange(current.AverageEfficiency, period2.AverageEfficiency),
		}
	}

	// Return nil if no comparisons were made
	if comparison.VsSamePeriodMinus1 == nil && comparison.VsSamePeriodMinus2 == nil {
		return nil
	}

	return comparison
}

// calculatePercentChange calculates the percentage change between two values
func calculatePercentChange(current, previous float64) float64 {
	if previous == 0 {
		return 0
	}
	return ((current - previous) / previous) * 100
}

// convertTimeSeries converts repository time series to response format
func convertTimeSeries(data []repository.TimeSeriesEntry) []TimeSeriesDataPoint {
	result := make([]TimeSeriesDataPoint, len(data))
	for i, entry := range data {
		result[i] = TimeSeriesDataPoint{
			Date:          entry.Date,
			NominalAmount: entry.NominalAmount,
			RealAmount:    entry.RealAmount,
			Efficiency:    entry.Efficiency,
			EventCount:    entry.EventCount,
		}
	}
	return result
}

// convertSectorBreakdown converts repository sector breakdown to response format
func convertSectorBreakdown(data []repository.SectorBreakdown) []SectorBreakdownData {
	result := make([]SectorBreakdownData, len(data))
	for i, entry := range data {
		result[i] = SectorBreakdownData{
			SectorID:          entry.SectorID,
			SectorName:        entry.SectorName,
			TotalVolume:       entry.TotalVolume,
			AverageEfficiency: entry.AverageEfficiency,
		}
	}
	return result
}

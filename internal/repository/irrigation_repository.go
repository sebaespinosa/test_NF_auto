package repository

import (
	"time"

	"github.com/sebaespinosa/test_NF_auto/internal/model"
	"gorm.io/gorm"
)

// IrrigationRepository defines the interface for irrigation data access
type IrrigationRepository interface {
	GetAnalytics(farmID uint, startDate, endDate time.Time, sectorID *uint) (*AnalyticsData, error)
	GetTimeSeriesData(farmID uint, startDate, endDate time.Time, sectorID *uint, aggregation string) ([]TimeSeriesEntry, error)
	GetSectorBreakdown(farmID uint, startDate, endDate time.Time) ([]SectorBreakdown, error)
}

// AnalyticsData represents aggregated analytics metrics
type AnalyticsData struct {
	TotalVolume       float64
	TotalEvents       int64
	AverageEfficiency float64
	MinEfficiency     float64
	MaxEfficiency     float64
}

// TimeSeriesEntry represents a single time-series data point
type TimeSeriesEntry struct {
	Date          string
	NominalAmount float64
	RealAmount    float64
	Efficiency    float64
	EventCount    int64
}

// SectorBreakdown represents aggregated data per sector
type SectorBreakdown struct {
	SectorID          uint
	SectorName        string
	TotalVolume       float64
	AverageEfficiency float64
}

type irrigationRepository struct {
	db *gorm.DB
}

// NewIrrigationRepository creates a new irrigation repository
func NewIrrigationRepository(db *gorm.DB) IrrigationRepository {
	return &irrigationRepository{db: db}
}

// GetAnalytics retrieves aggregated analytics for the specified period
func (r *irrigationRepository) GetAnalytics(farmID uint, startDate, endDate time.Time, sectorID *uint) (*AnalyticsData, error) {
	query := r.db.Model(&model.IrrigationData{}).
		Where("farm_id = ? AND start_time >= ? AND start_time < ?", farmID, startDate, endDate)

	if sectorID != nil {
		query = query.Where("irrigation_sector_id = ?", *sectorID)
	}

	var result struct {
		TotalVolume       float64
		TotalEvents       int64
		AverageEfficiency float64
		MinEfficiency     float64
		MaxEfficiency     float64
	}

	// Calculate metrics in a single query using database aggregations
	err := query.Select(`
		COALESCE(SUM(real_amount), 0) as total_volume,
		COUNT(*) as total_events,
		COALESCE(AVG(CASE WHEN nominal_amount > 0 THEN real_amount / nominal_amount ELSE NULL END), 0) as average_efficiency,
		COALESCE(MIN(CASE WHEN nominal_amount > 0 THEN real_amount / nominal_amount ELSE NULL END), 0) as min_efficiency,
		COALESCE(MAX(CASE WHEN nominal_amount > 0 THEN real_amount / nominal_amount ELSE NULL END), 0) as max_efficiency
	`).Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return &AnalyticsData{
		TotalVolume:       result.TotalVolume,
		TotalEvents:       result.TotalEvents,
		AverageEfficiency: result.AverageEfficiency,
		MinEfficiency:     result.MinEfficiency,
		MaxEfficiency:     result.MaxEfficiency,
	}, nil
}

// GetTimeSeriesData retrieves time-series data with the specified aggregation
func (r *irrigationRepository) GetTimeSeriesData(farmID uint, startDate, endDate time.Time, sectorID *uint, aggregation string) ([]TimeSeriesEntry, error) {
	query := r.db.Model(&model.IrrigationData{}).
		Where("farm_id = ? AND start_time >= ? AND start_time < ?", farmID, startDate, endDate)

	if sectorID != nil {
		query = query.Where("irrigation_sector_id = ?", *sectorID)
	}

	// Determine date truncation based on aggregation level
	var dateFormat string
	var dateTrunc string
	switch aggregation {
	case "weekly":
		dateFormat = "YYYY-\"W\"IW"
		dateTrunc = "week"
	case "monthly":
		dateFormat = "YYYY-MM"
		dateTrunc = "month"
	default: // daily
		dateFormat = "YYYY-MM-DD"
		dateTrunc = "day"
	}

	var results []struct {
		Date          string
		NominalAmount float64
		RealAmount    float64
		EventCount    int64
	}

	err := query.Select(`
		TO_CHAR(DATE_TRUNC(?, start_time), ?) as date,
		COALESCE(SUM(nominal_amount), 0) as nominal_amount,
		COALESCE(SUM(real_amount), 0) as real_amount,
		COUNT(*) as event_count
	`, dateTrunc, dateFormat).
		Group("date").
		Order("date").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	// Calculate efficiency for each entry
	entries := make([]TimeSeriesEntry, len(results))
	for i, r := range results {
		efficiency := 0.0
		if r.NominalAmount > 0 {
			efficiency = r.RealAmount / r.NominalAmount
		}
		entries[i] = TimeSeriesEntry{
			Date:          r.Date,
			NominalAmount: r.NominalAmount,
			RealAmount:    r.RealAmount,
			Efficiency:    efficiency,
			EventCount:    r.EventCount,
		}
	}

	return entries, nil
}

// GetSectorBreakdown retrieves aggregated data per sector
func (r *irrigationRepository) GetSectorBreakdown(farmID uint, startDate, endDate time.Time) ([]SectorBreakdown, error) {
	var results []SectorBreakdown

	err := r.db.Model(&model.IrrigationData{}).
		Select(`
			irrigation_data.irrigation_sector_id as sector_id,
			irrigation_sectors.name as sector_name,
			COALESCE(SUM(irrigation_data.real_amount), 0) as total_volume,
			COALESCE(AVG(CASE WHEN irrigation_data.nominal_amount > 0 THEN irrigation_data.real_amount / irrigation_data.nominal_amount ELSE NULL END), 0) as average_efficiency
		`).
		Joins("LEFT JOIN irrigation_sectors ON irrigation_sectors.id = irrigation_data.irrigation_sector_id").
		Where("irrigation_data.farm_id = ? AND irrigation_data.start_time >= ? AND irrigation_data.start_time < ?", farmID, startDate, endDate).
		Group("irrigation_data.irrigation_sector_id, irrigation_sectors.name").
		Order("irrigation_data.irrigation_sector_id").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	return results, nil
}

package model

import "time"

// IrrigationData represents irrigation event data
type IrrigationData struct {
	ID                 uint             `gorm:"primaryKey" json:"id"`
	FarmID             uint             `gorm:"not null;index:idx_farm_time" json:"farm_id"`
	IrrigationSectorID uint             `gorm:"not null;index:idx_sector_time" json:"irrigation_sector_id"`
	StartTime          time.Time        `gorm:"not null;index:idx_farm_time;index:idx_sector_time" json:"start_time"`
	EndTime            time.Time        `gorm:"not null" json:"end_time"`
	NominalAmount      float32          `gorm:"type:numeric(10,2)" json:"nominal_amount"` // in mm
	RealAmount         float32          `gorm:"type:numeric(10,2)" json:"real_amount"`    // in mm
	CreatedAt          time.Time        `json:"created_at"`
	UpdatedAt          time.Time        `json:"updated_at"`
	Farm               Farm             `gorm:"foreignKey:FarmID" json:"-"`
	IrrigationSector   IrrigationSector `gorm:"foreignKey:IrrigationSectorID" json:"-"`
}

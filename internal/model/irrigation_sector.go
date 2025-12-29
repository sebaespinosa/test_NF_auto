package model

import "time"

// IrrigationSector represents an irrigation sector within a farm
type IrrigationSector struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	FarmID    uint      `gorm:"not null;index" json:"farm_id"`
	Name      string    `gorm:"not null" json:"name"`
	Farm      Farm      `gorm:"foreignKey:FarmID" json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

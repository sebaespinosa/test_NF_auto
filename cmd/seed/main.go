package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/sebaespinosa/test_NF_auto/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
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

	log.Println("Connected to database, starting seed...")

	// Create random number generator
	randSource := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(randSource)

	// Create farms
	farms := []model.Farm{
		{Name: "Green Valley Farm"},
		{Name: "Sunshine Agriculture"},
		{Name: "Blue River Ranch"},
	}

	for i := range farms {
		result := db.Create(&farms[i])
		if result.Error != nil {
			log.Fatalf("Failed to create farm: %v", result.Error)
		}
		log.Printf("Created farm: %s (ID: %d)", farms[i].Name, farms[i].ID)
	}

	// Create irrigation sectors for each farm
	var sectors []model.IrrigationSector
	for _, farm := range farms {
		farmSectors := []model.IrrigationSector{
			{FarmID: farm.ID, Name: "Sector A"},
			{FarmID: farm.ID, Name: "Sector B"},
			{FarmID: farm.ID, Name: "Sector C"},
			{FarmID: farm.ID, Name: "Sector D"},
		}
		for i := range farmSectors {
			result := db.Create(&farmSectors[i])
			if result.Error != nil {
				log.Fatalf("Failed to create sector: %v", result.Error)
			}
			sectors = append(sectors, farmSectors[i])
		}
		log.Printf("Created 4 sectors for farm %s", farm.Name)
	}

	// Generate irrigation data spanning 3 years (2022, 2023, 2024)
	log.Println("Generating irrigation data...")

	eventCount := 0
	for year := 2022; year <= 2024; year++ {
		for month := 1; month <= 12; month++ {
			// Generate 3-5 events per month per sector
			for _, sector := range sectors {
				eventsPerMonth := rng.Intn(3) + 3 // 3-5 events

				for i := 0; i < eventsPerMonth; i++ {
					day := rng.Intn(28) + 1 // Days 1-28 to avoid month-end issues
					hour := rng.Intn(24)
					minute := rng.Intn(60)

					startTime := time.Date(year, time.Month(month), day, hour, minute, 0, 0, time.UTC)
					endTime := startTime.Add(time.Duration(rng.Intn(120)+30) * time.Minute) // 30-150 minutes

					// Generate realistic irrigation amounts (in mm)
					nominalAmount := float32(rng.Intn(20) + 5) // 5-25 mm
					// Real amount is usually 70-95% of nominal (simulating efficiency)
					efficiencyFactor := 0.7 + rng.Float32()*0.25
					realAmount := nominalAmount * efficiencyFactor

					irrigationData := model.IrrigationData{
						FarmID:             sector.FarmID,
						IrrigationSectorID: sector.ID,
						StartTime:          startTime,
						EndTime:            endTime,
						NominalAmount:      nominalAmount,
						RealAmount:         realAmount,
					}

					result := db.Create(&irrigationData)
					if result.Error != nil {
						log.Printf("Failed to create irrigation data: %v", result.Error)
						continue
					}
					eventCount++
				}
			}
		}
		log.Printf("Generated irrigation data for year %d", year)
	}

	log.Printf("Seed completed! Created %d farms, %d sectors, and %d irrigation events", len(farms), len(sectors), eventCount)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

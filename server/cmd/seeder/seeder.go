// Package seeder provides functionality for populating the database
// with initial mock data and checking database state.
package seeder

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"gorm.io/gorm"
)

const successStatus = "success"

// User represents a system user entity in the database.
type User struct {
	ID        uint        `gorm:"primaryKey"`
	Username  string      `gorm:"type:varchar(50);not null;unique"`
	Email     string      `gorm:"type:varchar(100);not null"`
	CreatedAt time.Time   `gorm:"default:now()"`
	Logs      []SystemLog `gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL"`
}

// AppSetting represents system-wide key-value configuration settings.
type AppSetting struct {
	Key       string    `gorm:"type:varchar(50);primaryKey"`
	Value     string    `gorm:"type:text;not null"`
	UpdatedAt time.Time `gorm:"default:now()"`
}

// SystemLog represents audit logs for system operations and user actions.
type SystemLog struct {
	ID        uint            `gorm:"primaryKey"`
	UserID    *uint           `gorm:"index"`
	Action    string          `gorm:"type:varchar(100);not null"`
	Status    string          `gorm:"type:varchar(20);not null"`
	Payload   json.RawMessage `gorm:"type:jsonb"`
	CreatedAt time.Time       `gorm:"default:now()"`
}

// DBSeeder drops existing tables, migrates schemas, and populates the database with test data.
func DBSeeder(db *gorm.DB) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	log.Println("Dropping old tables if they exist...")
	err := db.Migrator().DropTable(&SystemLog{}, &AppSetting{}, &User{})
	if err != nil {
		log.Fatalf("Failed to drop tables: %v", err)
	}

	log.Println("Running AutoMigrate...")
	err = db.AutoMigrate(&User{}, &AppSetting{}, &SystemLog{})
	if err != nil {
		log.Fatalf("Failed to auto-migrate schemas: %v", err)
	}
	log.Println("Tables created successfully.")

	log.Println("Inserting mock data...")

	settings := []AppSetting{
		{Key: "maintenance_mode", Value: "false"},
		{Key: "max_connections", Value: "100"},
		{Key: "api_version", Value: "v1.0.2"},
	}
	if err := db.Create(&settings).Error; err != nil {
		log.Fatalf("Failed to insert settings: %v", err)
	}

	mockUsers := []User{
		{Username: "admin", Email: "admin@dbsentinel.io"},
		{Username: "operator_alex", Email: "alex@dbsentinel.io"},
		{Username: "dev_john", Email: "john@dbsentinel.io"},
		{Username: "guest_user", Email: "guest@dbsentinel.io"},
	}
	if err := db.Create(&mockUsers).Error; err != nil {
		log.Fatalf("Failed to insert users: %v", err)
	}

	var userIDs []uint
	for _, u := range mockUsers {
		userIDs = append(userIDs, u.ID)
	}

	actions := []string{"user.login", "user.logout", "backup.triggered", "settings.update", "topology.check"}
	statuses := []string{successStatus, successStatus, "failed", successStatus}
	var mockLogs []SystemLog

	for i := 0; i < 50; i++ {
		// Используем созданный экземпляр рандома r
		uid := userIDs[r.Intn(len(userIDs))]
		action := actions[r.Intn(len(actions))]
		status := statuses[r.Intn(len(statuses))]

		rawJSON := fmt.Sprintf(`{"meta": "log_record_%d", "execution_time_ms": %d}`, i, r.Intn(150))

		mockLogs = append(mockLogs, SystemLog{
			UserID:    &uid,
			Action:    action,
			Status:    status,
			Payload:   json.RawMessage(rawJSON),
			CreatedAt: time.Now().Add(time.Duration(-r.Intn(24)) * time.Hour),
		})
	}

	if err := db.Create(&mockLogs).Error; err != nil {
		log.Fatalf("Failed to insert system logs: %v", err)
	}

	log.Printf("Seeding complete! Generated %d users, %d settings, and %d system logs.\n",
		len(mockUsers), len(settings), len(mockLogs),
	)
}

// IsDatabaseEmpty checks if the database needs seeding by verifying table existence and user count.
func IsDatabaseEmpty(db *gorm.DB) bool {
	if !db.Migrator().HasTable(&User{}) {
		return true
	}

	var count int64
	if err := db.Model(&User{}).Count(&count).Error; err != nil {
		log.Printf("Warning: failed to count users: %v. Assuming database needs seeding.", err)
		return true
	}

	return count == 0
}

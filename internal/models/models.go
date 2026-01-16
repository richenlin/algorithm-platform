package models

import (
	"time"

	"gorm.io/gorm"
)

type Algorithm struct {
	ID               string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	Name             string    `gorm:"type:varchar(255);not null" json:"name"`
	Description      string    `gorm:"type:text" json:"description"`
	Language         string    `gorm:"type:varchar(50)" json:"language"`
	Platform         string    `gorm:"type:varchar(50)" json:"platform"`
	Category         string    `gorm:"type:varchar(255)" json:"category"`
	Entrypoint       string    `gorm:"type:varchar(255)" json:"entrypoint"`
	Tags             string    `gorm:"type:text" json:"tags"`
	PresetDataID     string    `gorm:"type:varchar(36)" json:"preset_data_id"`
	CurrentVersionID string    `gorm:"type:varchar(36)" json:"current_version_id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	Versions []Version `gorm:"foreignKey:AlgorithmID" json:"versions,omitempty"`
}

type Version struct {
	ID             string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	AlgorithmID    string    `gorm:"type:varchar(36);not null" json:"algorithm_id"`
	VersionNumber  int       `gorm:"not null" json:"version_number"`
	MinioPath      string    `gorm:"type:text" json:"minio_path"`
	SourceCodeFile string    `gorm:"type:text" json:"source_code_file"`
	CommitMessage  string    `gorm:"type:text" json:"commit_message"`
	CreatedAt      time.Time `json:"created_at"`

	Algorithm Algorithm `gorm:"foreignKey:AlgorithmID" json:"algorithm,omitempty"`
}

type Job struct {
	ID            string     `gorm:"primaryKey;type:varchar(36)" json:"job_id"`
	AlgorithmID   string     `gorm:"type:varchar(36);index" json:"algorithm_id"`
	AlgorithmName string     `gorm:"type:varchar(255)" json:"algorithm_name"`
	Mode          string     `gorm:"type:varchar(50)" json:"mode"`
	Status        string     `gorm:"type:varchar(50);index" json:"status"`
	InputParams   string     `gorm:"type:text" json:"input_params"`
	InputURL      string     `gorm:"type:text" json:"input_url"`
	OutputURL     string     `gorm:"type:text" json:"output_url"`
	LogURL        string     `gorm:"type:text" json:"log_url"`
	StartedAt     *time.Time `json:"started_at"`
	FinishedAt    *time.Time `json:"finished_at"`
	CostTimeMs    int64      `json:"cost_time_ms"`
	WorkerID      string     `gorm:"type:varchar(36)" json:"worker_id"`
	CreatedAt     time.Time  `json:"created_at"`
}

type PresetData struct {
	ID        string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	Filename  string    `gorm:"type:varchar(255);not null" json:"filename"`
	Category  string    `gorm:"type:varchar(255);index" json:"category"`
	MinioURL  string    `gorm:"type:text" json:"minio_url"`
	CreatedAt time.Time `json:"created_at"`
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Algorithm{},
		&Version{},
		&Job{},
		&PresetData{},
	)
}

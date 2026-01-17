package models

import (
	"time"

	"gorm.io/gorm"
)

// DatabaseMetadata 数据库元数据，用于版本控制和数据同步
type DatabaseMetadata struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Version       int64     `gorm:"not null;index" json:"version"`         // 数据版本号，每次写入递增
	LastUpdatedAt time.Time `gorm:"not null;index" json:"last_updated_at"` // 最后更新时间
	UpdatedBy     string    `gorm:"type:varchar(100)" json:"updated_by"`   // 更新来源（如：api, backup_restore）
	CheckpointAt  time.Time `json:"checkpoint_at"`                         // 最后checkpoint时间
	RecordCount   int64     `json:"record_count"`                          // 总记录数
}

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
	MinioPath string    `gorm:"type:text" json:"minio_path"` // MinIO路径
	MinioURL  string    `gorm:"type:text" json:"minio_url"`  // 完整URL（已废弃，保留兼容性）
	CreatedAt time.Time `json:"created_at"`
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&DatabaseMetadata{},
		&Algorithm{},
		&Version{},
		&Job{},
		&PresetData{},
	)
}

func (Job) TableName() string {
	return "jobs"
}

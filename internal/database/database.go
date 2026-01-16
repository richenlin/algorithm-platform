package database

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"algorithm-platform/internal/config"
	"algorithm-platform/internal/models"

	"github.com/mattn/go-sqlite3"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Database struct {
	db         *gorm.DB
	minio      *minio.Client
	bucketName string
	cfg        *config.Config
}

func New(cfg *config.Config) (*Database, error) {
	minioClient, err := minio.New(cfg.MinIO.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIO.AccessKeyID, cfg.MinIO.SecretAccessKey, ""),
		Secure: cfg.MinIO.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MinIO client: %w", err)
	}

	dataDir := "./data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "algorithm-platform.db")
	db, err := gorm.Open(sqlite.Dialector{
		DSN: dbPath,
	}, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	dbDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}
	dbDB.SetMaxOpenConns(1)
	dbDB.SetMaxIdleConns(1)

	if err := models.AutoMigrate(db); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	database := &Database{
		db:         db,
		minio:      minioClient,
		bucketName: cfg.MinIO.Bucket,
		cfg:        cfg,
	}

	if err := database.loadFromMinIO(context.Background()); err != nil {
		fmt.Printf("Warning: failed to load data from MinIO: %v\n", err)
	}

	if err := database.startBackupScheduler(context.Background()); err != nil {
		fmt.Printf("Warning: failed to start backup scheduler: %v\n", err)
	}

	return database, nil
}

func (d *Database) DB() *gorm.DB {
	return d.db
}

func (d *Database) MinIO() *minio.Client {
	return d.minio
}

func (d *Database) loadFromMinIO(ctx context.Context) error {
	exists, err := d.minio.BucketExists(ctx, d.bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if !exists {
		return d.minio.MakeBucket(ctx, d.bucketName, minio.MakeBucketOptions{})
	}

	backupPath := "database-backup/latest.json"
	obj, err := d.minio.GetObject(ctx, d.bucketName, backupPath, minio.GetObjectOptions{})
	if err != nil {
		return nil
	}
	defer obj.Close()

	var backupData map[string]interface{}
	if err := json.NewDecoder(obj).Decode(&backupData); err != nil {
		return fmt.Errorf("failed to decode backup: %w", err)
	}

	if algorithms, ok := backupData["algorithms"].([]interface{}); ok {
		for _, alg := range algorithms {
			if algMap, ok := alg.(map[string]interface{}); ok {
				var algorithm models.Algorithm
				algorithmData, _ := json.Marshal(algMap)
				json.Unmarshal(algorithmData, &algorithm)
				if result := d.db.FirstOrCreate(&algorithm, "id = ?", algorithm.ID); result.Error != nil {
					fmt.Printf("Failed to restore algorithm %s: %v\n", algorithm.ID, result.Error)
				}
			}
		}
	}

	if presetData, ok := backupData["preset_data"].([]interface{}); ok {
		for _, data := range presetData {
			if dataMap, ok := data.(map[string]interface{}); ok {
				var presetData models.PresetData
				dataData, _ := json.Marshal(dataMap)
				json.Unmarshal(dataData, &presetData)
				if result := d.db.FirstOrCreate(&presetData, "id = ?", presetData.ID); result.Error != nil {
					fmt.Printf("Failed to restore preset data %s: %v\n", presetData.ID, result.Error)
				}
			}
		}
	}

	fmt.Println("Data loaded from MinIO backup")
	return nil
}

func (d *Database) backupToMinIO(ctx context.Context) error {
	var algorithms []models.Algorithm
	if err := d.db.Find(&algorithms).Error; err != nil {
		return fmt.Errorf("failed to fetch algorithms: %w", err)
	}

	var versions []models.Version
	if err := d.db.Find(&versions).Error; err != nil {
		return fmt.Errorf("failed to fetch versions: %w", err)
	}

	for i := range algorithms {
		if err := d.db.Model(&algorithms[i]).Association("Versions").Find(&algorithms[i].Versions); err != nil {
			fmt.Printf("Failed to load versions for algorithm %s: %v\n", algorithms[i].ID, err)
		}
	}

	var presetData []models.PresetData
	if err := d.db.Find(&presetData).Error; err != nil {
		return fmt.Errorf("failed to fetch preset data: %w", err)
	}

	var jobs []models.Job
	if err := d.db.Find(&jobs).Error; err != nil {
		return fmt.Errorf("failed to fetch jobs: %w", err)
	}

	backupData := map[string]interface{}{
		"algorithms":  algorithms,
		"versions":    versions,
		"preset_data": presetData,
		"jobs":        jobs,
		"backuped_at": time.Now(),
	}

	backupJSON, err := json.Marshal(backupData)
	if err != nil {
		return fmt.Errorf("failed to marshal backup data: %w", err)
	}

	backupPath := fmt.Sprintf("database-backup/backup-%s.json", time.Now().Format("20060102-150405"))
	_, err = d.minio.PutObject(ctx, d.bucketName, backupPath, nil, int64(len(backupJSON)), minio.PutObjectOptions{
		ContentType: "application/json",
	})
	if err != nil {
		return fmt.Errorf("failed to upload backup to MinIO: %w", err)
	}

	latestPath := "database-backup/latest.json"
	_, err = d.minio.PutObject(ctx, d.bucketName, latestPath, nil, int64(len(backupJSON)), minio.PutObjectOptions{
		ContentType: "application/json",
	})
	if err != nil {
		return fmt.Errorf("failed to update latest backup: %w", err)
	}

	fmt.Printf("Backup saved to MinIO: %s\n", backupPath)
	return nil
}

func (d *Database) startBackupScheduler(ctx context.Context) error {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := d.backupToMinIO(ctx); err != nil {
					fmt.Printf("Backup failed: %v\n", err)
				}
			}
		}
	}()

	return nil
}

func (d *Database) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}

	ctx := context.Background()
	if err := d.backupToMinIO(ctx); err != nil {
		fmt.Printf("Final backup failed: %v\n", err)
	}

	return sqlDB.Close()
}

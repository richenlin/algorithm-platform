package database

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"algorithm-platform/internal/config"
	"algorithm-platform/internal/models"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gorm.io/gorm"
)

// SQLiteBackupManager SQLite 专用的备份管理器
// 负责将 SQLite 数据备份到 MinIO 和本地文件系统
type SQLiteBackupManager struct {
	db             *gorm.DB
	minio          *minio.Client
	bucketName     string
	stopBackup     chan struct{}
	backupInterval time.Duration
}

// NewSQLiteBackupManager 创建 SQLite 备份管理器
func NewSQLiteBackupManager(db *gorm.DB, cfg *config.Config, bucketName string) *SQLiteBackupManager {
	minioClient, err := minio.New(cfg.MinIO.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIO.AccessKeyID, cfg.MinIO.SecretAccessKey, ""),
		Secure: cfg.MinIO.UseSSL,
	})
	if err != nil {
		fmt.Errorf("failed to initialize MinIO client: %w", err)
	}
	return &SQLiteBackupManager{
		db:             db,
		minio:          minioClient,
		bucketName:     bucketName,
		stopBackup:     make(chan struct{}),
		backupInterval: 5 * time.Minute,
	}
}

// LoadFromMinIO 从 MinIO 加载备份数据
func (m *SQLiteBackupManager) LoadFromMinIO(ctx context.Context) error {
	exists, err := m.minio.BucketExists(ctx, m.bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if !exists {
		return m.minio.MakeBucket(ctx, m.bucketName, minio.MakeBucketOptions{})
	}

	backupPath := "database-backup/latest.json"
	obj, err := m.minio.GetObject(ctx, m.bucketName, backupPath, minio.GetObjectOptions{})
	if err != nil {
		return nil
	}
	defer obj.Close()

	var backupData map[string]interface{}
	if err := json.NewDecoder(obj).Decode(&backupData); err != nil {
		return fmt.Errorf("failed to decode backup: %w", err)
	}

	// 恢复算法数据
	if algorithms, ok := backupData["algorithms"].([]interface{}); ok {
		for _, alg := range algorithms {
			if algMap, ok := alg.(map[string]interface{}); ok {
				var algorithm models.Algorithm
				algorithmData, _ := json.Marshal(algMap)
				json.Unmarshal(algorithmData, &algorithm)
				if result := m.db.FirstOrCreate(&algorithm, "id = ?", algorithm.ID); result.Error != nil {
					fmt.Printf("Failed to restore algorithm %s: %v\n", algorithm.ID, result.Error)
				}
			}
		}
	}

	// 恢复预设数据
	if presetData, ok := backupData["preset_data"].([]interface{}); ok {
		for _, data := range presetData {
			if dataMap, ok := data.(map[string]interface{}); ok {
				var presetData models.PresetData
				dataData, _ := json.Marshal(dataMap)
				json.Unmarshal(dataData, &presetData)
				if result := m.db.FirstOrCreate(&presetData, "id = ?", presetData.ID); result.Error != nil {
					fmt.Printf("Failed to restore preset data %s: %v\n", presetData.ID, result.Error)
				}
			}
		}
	}

	fmt.Println("SQLite data loaded from MinIO backup")
	return nil
}

// BackupToMinIO 备份数据到 MinIO 和本地
func (m *SQLiteBackupManager) BackupToMinIO(ctx context.Context) error {
	// 获取所有数据
	var algorithms []models.Algorithm
	if err := m.db.Find(&algorithms).Error; err != nil {
		return fmt.Errorf("failed to fetch algorithms: %w", err)
	}

	var versions []models.Version
	if err := m.db.Find(&versions).Error; err != nil {
		return fmt.Errorf("failed to fetch versions: %w", err)
	}

	for i := range algorithms {
		if err := m.db.Model(&algorithms[i]).Association("Versions").Find(&algorithms[i].Versions); err != nil {
			fmt.Printf("Failed to load versions for algorithm %s: %v\n", algorithms[i].ID, err)
		}
	}

	var presetData []models.PresetData
	if err := m.db.Find(&presetData).Error; err != nil {
		return fmt.Errorf("failed to fetch preset data: %w", err)
	}

	var jobs []models.Job
	if err := m.db.Find(&jobs).Error; err != nil {
		return fmt.Errorf("failed to fetch jobs: %w", err)
	}

	backupData := map[string]interface{}{
		"algorithms":  algorithms,
		"versions":    versions,
		"preset_data": presetData,
		"jobs":        jobs,
		"backuped_at": time.Now(),
		"backup_type": "sqlite",
	}

	backupJSON, err := json.MarshalIndent(backupData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal backup data: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")

	// 1. 保存本地备份
	if err := m.saveLocalBackup(backupJSON, timestamp); err != nil {
		fmt.Printf("Warning: local backup failed: %v\n", err)
	}

	// 2. 上传到 MinIO（带时间戳的备份）
	backupPath := fmt.Sprintf("database-backup/backup-%s.json", timestamp)
	_, err = m.minio.PutObject(ctx, m.bucketName, backupPath,
		bytes.NewReader(backupJSON), int64(len(backupJSON)),
		minio.PutObjectOptions{
			ContentType: "application/json",
		})
	if err != nil {
		return fmt.Errorf("failed to upload backup to MinIO: %w", err)
	}

	// 3. 更新 latest 备份
	latestPath := "database-backup/latest.json"
	_, err = m.minio.PutObject(ctx, m.bucketName, latestPath,
		bytes.NewReader(backupJSON), int64(len(backupJSON)),
		minio.PutObjectOptions{
			ContentType: "application/json",
		})
	if err != nil {
		return fmt.Errorf("failed to update latest backup: %w", err)
	}

	fmt.Printf("SQLite backup saved: MinIO=%s, Local=%s\n", backupPath, timestamp)

	// 4. 异步清理旧备份
	go m.cleanupOldBackups(ctx)

	return nil
}

// saveLocalBackup 保存本地备份
func (m *SQLiteBackupManager) saveLocalBackup(data []byte, timestamp string) error {
	backupDir := "./data/backups"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	backupFile := filepath.Join(backupDir, fmt.Sprintf("backup-%s.json", timestamp))
	if err := os.WriteFile(backupFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	fmt.Printf("Local backup saved: %s\n", backupFile)
	return nil
}

// cleanupOldBackups 清理旧备份
func (m *SQLiteBackupManager) cleanupOldBackups(ctx context.Context) {
	// 清理 MinIO 旧备份（保留最近 10 个）
	objectCh := m.minio.ListObjects(ctx, m.bucketName, minio.ListObjectsOptions{
		Prefix:    "database-backup/backup-",
		Recursive: true,
	})

	var backups []string
	for object := range objectCh {
		if object.Err != nil {
			fmt.Printf("Error listing backups: %v\n", object.Err)
			return
		}
		if object.Key != "database-backup/latest.json" {
			backups = append(backups, object.Key)
		}
	}

	sort.Strings(backups)

	if len(backups) > 10 {
		for _, key := range backups[:len(backups)-10] {
			if err := m.minio.RemoveObject(ctx, m.bucketName, key, minio.RemoveObjectOptions{}); err != nil {
				fmt.Printf("Failed to delete old backup %s: %v\n", key, err)
			} else {
				fmt.Printf("Deleted old MinIO backup: %s\n", key)
			}
		}
	}

	// 清理本地旧备份（保留最近 5 个）
	m.cleanupLocalBackups()
}

// cleanupLocalBackups 清理本地旧备份
func (m *SQLiteBackupManager) cleanupLocalBackups() {
	backupDir := "./data/backups"
	files, err := filepath.Glob(filepath.Join(backupDir, "backup-*.json"))
	if err != nil {
		return
	}

	sort.Strings(files)

	if len(files) > 5 {
		for _, file := range files[:len(files)-5] {
			if err := os.Remove(file); err != nil {
				fmt.Printf("Failed to delete local backup %s: %v\n", file, err)
			} else {
				fmt.Printf("Deleted old local backup: %s\n", file)
			}
		}
	}
}

// StartBackupScheduler 启动备份调度器
func (m *SQLiteBackupManager) StartBackupScheduler(ctx context.Context) error {
	ticker := time.NewTicker(m.backupInterval)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-m.stopBackup:
				return
			case <-ticker.C:
				if err := m.BackupToMinIO(context.Background()); err != nil {
					fmt.Printf("SQLite backup failed: %v\n", err)
				}
			}
		}
	}()

	fmt.Printf("SQLite backup scheduler started (interval: %v)\n", m.backupInterval)
	return nil
}

// Stop 停止备份调度器
func (m *SQLiteBackupManager) Stop() {
	close(m.stopBackup)
	fmt.Println("SQLite backup scheduler stopped")
}

// SetBackupInterval 设置备份间隔
func (m *SQLiteBackupManager) SetBackupInterval(interval time.Duration) {
	m.backupInterval = interval
}

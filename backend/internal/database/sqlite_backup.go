package database

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
	dbPath         string // 数据库文件路径
}

// NewSQLiteBackupManager 创建 SQLite 备份管理器
func NewSQLiteBackupManager(db *gorm.DB, cfg *config.Config) (*SQLiteBackupManager, error) {
	// 初始化 MinIO 客户端
	minioClient, err := minio.New(cfg.MinIO.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIO.AccessKeyID, cfg.MinIO.SecretAccessKey, ""),
		Secure: cfg.MinIO.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MinIO client: %w", err)
	}

	return &SQLiteBackupManager{
		db:             db,
		minio:          minioClient,
		bucketName:     cfg.MinIO.Bucket,
		stopBackup:     make(chan struct{}),
		backupInterval: 5 * time.Minute,
		dbPath:         cfg.Database.SQLite.Path,
	}, nil
}

// BackupMetadata 备份元数据
type BackupMetadata struct {
	Timestamp     time.Time `json:"timestamp"`
	Hash          string    `json:"hash"`
	Source        string    `json:"source"` // "minio" or "local"
	Path          string    `json:"path"`
	Version       int64     `json:"version"`         // 数据版本号
	RecordCount   int64     `json:"record_count"`    // 记录数量
	LastUpdatedAt time.Time `json:"last_updated_at"` // 数据最后更新时间
}

// LoadFromMinIO 智能恢复策略：使用版本号比对，选择最新数据
func (m *SQLiteBackupManager) LoadFromMinIO() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	fmt.Println("\n🔍 Checking database status...")

	// 获取当前数据库的元数据
	currentMeta, err := m.getDatabaseMetadata()
	if err != nil {
		fmt.Printf("❌ Failed to read database metadata: %v\n", err)
		fmt.Println("⚠️  Database may be corrupted or uninitialized")

		// 尝试从备份恢复
		if restoreErr := m.attemptRestore(ctx); restoreErr != nil {
			fmt.Println("💡 MANUAL ACTION REQUIRED:")
			fmt.Println("   1. Check if database file is corrupted: ", m.dbPath)
			fmt.Println("   2. Try restoring from backup manually")
			fmt.Println("   3. Contact system administrator if issue persists")
			return fmt.Errorf("failed to restore database: %w", restoreErr)
		}
		return nil
	}

	// 检查是否有数据
	if currentMeta.RecordCount == 0 {
		fmt.Println("⚠️  Database is empty (0 records)")

		// 获取可用备份
		minioBackup, _ := m.getMinIOBackupMetadata(ctx)
		localBackup, _ := m.getLocalBackupMetadata()

		if minioBackup == nil && localBackup == nil {
			fmt.Println("ℹ️  No backups found - this is first startup")
			fmt.Println("✅ Starting with empty database")
			return nil
		}

		// 有备份可用，从最新的恢复
		fmt.Println("ℹ️  Backups available, will restore from latest")
		return m.attemptRestore(ctx)
	}

	// 数据库有数据，比较版本号
	fmt.Printf("✅ Database has data (version: %d, records: %d, last_update: %s)\n",
		currentMeta.Version,
		currentMeta.RecordCount,
		currentMeta.LastUpdatedAt.Format("2006-01-02 15:04:05"))

	// 获取备份元数据
	minioBackup, err := m.getMinIOBackupMetadata(ctx)
	if err != nil {
		fmt.Printf("ℹ️  No MinIO backup found: %v\n", err)
	}

	localBackup, err := m.getLocalBackupMetadata()
	if err != nil {
		fmt.Printf("ℹ️  No local backup found: %v\n", err)
	}

	// 如果没有任何备份，保留现有数据
	if minioBackup == nil && localBackup == nil {
		fmt.Println("✅ No backups found, keeping current database")
		return nil
	}

	// 选择版本号最大的数据源
	newestSource := "current"
	newestVersion := currentMeta.Version
	newestTime := currentMeta.LastUpdatedAt
	var newestBackup *BackupMetadata

	if minioBackup != nil {
		fmt.Printf("   MinIO backup: version=%d, records=%d, time=%s\n",
			minioBackup.Version, minioBackup.RecordCount, minioBackup.LastUpdatedAt.Format("2006-01-02 15:04:05"))

		if minioBackup.Version > newestVersion ||
			(minioBackup.Version == newestVersion && minioBackup.LastUpdatedAt.After(newestTime)) {
			newestSource = "minio"
			newestVersion = minioBackup.Version
			newestTime = minioBackup.LastUpdatedAt
			newestBackup = minioBackup
		}
	}

	if localBackup != nil {
		fmt.Printf("   Local backup: version=%d, records=%d, time=%s\n",
			localBackup.Version, localBackup.RecordCount, localBackup.LastUpdatedAt.Format("2006-01-02 15:04:05"))

		if localBackup.Version > newestVersion ||
			(localBackup.Version == newestVersion && localBackup.LastUpdatedAt.After(newestTime)) {
			newestSource = "local"
			newestVersion = localBackup.Version
			newestTime = localBackup.LastUpdatedAt
			newestBackup = localBackup
		}
	}

	// 判断是否需要恢复
	if newestSource == "current" {
		fmt.Printf("✅ Current database is newest (version: %d)\n", currentMeta.Version)
		return nil
	}

	// 备份更新，执行恢复
	fmt.Println("\n⚠️  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("⚠️  BACKUP IS NEWER (version %d > %d)\n", newestVersion, currentMeta.Version)
	fmt.Println("⚠️  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("   Current:     version=%d, records=%d, time=%s\n",
		currentMeta.Version, currentMeta.RecordCount, currentMeta.LastUpdatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("   %s backup: version=%d, records=%d, time=%s\n",
		newestSource, newestVersion, newestBackup.RecordCount, newestTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("   Action: Will restore from %s backup\n", newestSource)

	// 执行恢复
	restoreChan := make(chan error, 1)
	go func() {
		restoreChan <- m.restoreFromBackup(ctx, newestBackup)
	}()

	select {
	case err := <-restoreChan:
		if err != nil {
			fmt.Println("\n❌ RESTORE FAILED")
			fmt.Printf("   Error: %v\n", err)
			fmt.Println("   ⚠️  Keeping current database")
			return nil // 不中断启动，保留当前数据
		}
		fmt.Println("✅ Database restored successfully")
	case <-ctx.Done():
		fmt.Println("\n❌ RESTORE TIMEOUT (exceeded 5 minutes)")
		fmt.Println("   ⚠️  Keeping current database")
		return nil // 不中断启动
	}

	return nil
}

// getDatabaseMetadata 获取当前数据库的元数据
func (m *SQLiteBackupManager) getDatabaseMetadata() (*BackupMetadata, error) {
	var meta models.DatabaseMetadata

	// 尝试获取最新的元数据记录
	if err := m.db.Order("version DESC").First(&meta).Error; err != nil {
		// 如果表不存在或没有记录，返回默认值
		if err == gorm.ErrRecordNotFound || isTableNotExistError(err) {
			// 统计实际记录数
			var count int64
			if err := m.db.Model(&models.Algorithm{}).Count(&count).Error; err != nil {
				// 如果 algorithms 表也不存在，说明数据库刚初始化
				if isTableNotExistError(err) {
					return &BackupMetadata{
						Version:       0,
						RecordCount:   0,
						LastUpdatedAt: time.Time{},
						Source:        "current",
					}, nil
				}
				return nil, err
			}

			// 返回默认元数据
			return &BackupMetadata{
				Version:       0,
				RecordCount:   count,
				LastUpdatedAt: time.Time{},
				Source:        "current",
			}, nil
		}
		return nil, err
	}

	// 统计当前记录数
	var count int64
	if err := m.db.Model(&models.Algorithm{}).Count(&count).Error; err != nil {
		return nil, err
	}

	return &BackupMetadata{
		Version:       meta.Version,
		RecordCount:   count,
		LastUpdatedAt: meta.LastUpdatedAt,
		Source:        "current",
	}, nil
}

// isTableNotExistError 检查是否是表不存在错误
func isTableNotExistError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "no such table") || strings.Contains(errStr, "doesn't exist")
}

// updateDatabaseMetadata 更新数据库元数据（每次写操作后调用）
func (m *SQLiteBackupManager) updateDatabaseMetadata(updatedBy string) error {
	var count int64
	if err := m.db.Model(&models.Algorithm{}).Count(&count).Error; err != nil {
		return err
	}

	// 获取当前最大版本号
	var currentMeta models.DatabaseMetadata
	m.db.Order("version DESC").First(&currentMeta)

	newMeta := models.DatabaseMetadata{
		Version:       currentMeta.Version + 1, // 版本号递增
		LastUpdatedAt: time.Now(),
		UpdatedBy:     updatedBy,
		CheckpointAt:  time.Now(),
		RecordCount:   count,
	}

	return m.db.Create(&newMeta).Error
}

// attemptRestore 尝试从备份恢复（仅在数据库损坏时调用）
func (m *SQLiteBackupManager) attemptRestore(ctx context.Context) error {
	fmt.Println("🔍 Looking for backups to restore...")

	// 确保 bucket 存在
	exists, err := m.minio.BucketExists(ctx, m.bucketName)
	if err != nil {
		fmt.Printf("❌ Failed to check MinIO bucket: %v\n", err)
		return err
	}
	if !exists {
		if err := m.minio.MakeBucket(ctx, m.bucketName, minio.MakeBucketOptions{}); err != nil {
			fmt.Printf("⚠️  Failed to create bucket: %v\n", err)
		} else {
			fmt.Printf("✅ Created MinIO bucket: %s\n", m.bucketName)
		}
		return fmt.Errorf("no backup available")
	}

	// 获取MinIO备份
	minioBackup, err := m.getMinIOBackupMetadata(ctx)
	if err != nil {
		fmt.Printf("ℹ️  No MinIO backup found: %v\n", err)
	}

	// 获取本地备份
	localBackup, err := m.getLocalBackupMetadata()
	if err != nil {
		fmt.Printf("ℹ️  No local backup found: %v\n", err)
	}

	// 选择最新的备份
	var newestBackup *BackupMetadata
	if minioBackup != nil && localBackup != nil {
		if minioBackup.Timestamp.After(localBackup.Timestamp) {
			newestBackup = minioBackup
		} else {
			newestBackup = localBackup
		}
	} else if minioBackup != nil {
		newestBackup = minioBackup
	} else if localBackup != nil {
		newestBackup = localBackup
	}

	if newestBackup == nil {
		return fmt.Errorf("no backup available")
	}

	// 恢复数据
	fmt.Printf("\n🔄 Restoring from %s backup\n", newestBackup.Source)
	fmt.Printf("   Time: %s\n", newestBackup.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("   Hash: %s\n", newestBackup.Hash[:16])

	restoreChan := make(chan error, 1)
	go func() {
		restoreChan <- m.restoreFromBackup(ctx, newestBackup)
	}()

	select {
	case err := <-restoreChan:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return fmt.Errorf("restore timeout exceeded 5 minutes")
	}

	return nil
}

// calculateDatabaseHash 计算当前数据库内容的hash
func (m *SQLiteBackupManager) calculateDatabaseHash() (string, error) {
	var algorithms []models.Algorithm
	if err := m.db.Find(&algorithms).Error; err != nil {
		return "", fmt.Errorf("failed to fetch algorithms: %w", err)
	}

	var presetData []models.PresetData
	if err := m.db.Find(&presetData).Error; err != nil {
		return "", fmt.Errorf("failed to fetch preset data: %w", err)
	}

	// 创建JSON表示
	data := map[string]interface{}{
		"algorithms":  algorithms,
		"preset_data": presetData,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal data: %w", err)
	}

	// 计算SHA256
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:]), nil
}

// getMinIOBackupMetadata 获取MinIO备份的元数据
func (m *SQLiteBackupManager) getMinIOBackupMetadata(ctx context.Context) (*BackupMetadata, error) {
	backupPath := "database-backup/latest.json"

	// 检查对象是否存在
	stat, err := m.minio.StatObject(ctx, m.bucketName, backupPath, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("backup not found: %w", err)
	}

	// 获取备份内容
	obj, err := m.minio.GetObject(ctx, m.bucketName, backupPath, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get backup: %w", err)
	}
	defer obj.Close()

	// 读取内容并计算hash
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(obj); err != nil {
		return nil, fmt.Errorf("failed to read backup: %w", err)
	}

	hash := sha256.Sum256(buf.Bytes())

	// 解析备份内容以获取元数据
	var backupData map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &backupData); err != nil {
		return nil, fmt.Errorf("failed to parse backup: %w", err)
	}

	// 提取元数据
	version := int64(0)
	recordCount := int64(0)
	lastUpdatedAt := stat.LastModified

	if metadata, ok := backupData["metadata"].(map[string]interface{}); ok {
		if v, ok := metadata["version"].(float64); ok {
			version = int64(v)
		}
		if rc, ok := metadata["record_count"].(float64); ok {
			recordCount = int64(rc)
		}
		if luat, ok := metadata["last_updated_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, luat); err == nil {
				lastUpdatedAt = t
			}
		}
	}

	// 如果没有元数据，尝试从算法数量估算
	if recordCount == 0 {
		if algorithms, ok := backupData["algorithms"].([]interface{}); ok {
			recordCount = int64(len(algorithms))
		}
	}

	return &BackupMetadata{
		Timestamp:     stat.LastModified,
		Hash:          hex.EncodeToString(hash[:]),
		Source:        "minio",
		Path:          backupPath,
		Version:       version,
		RecordCount:   recordCount,
		LastUpdatedAt: lastUpdatedAt,
	}, nil
}

// getLocalBackupMetadata 获取本地最新备份的元数据
func (m *SQLiteBackupManager) getLocalBackupMetadata() (*BackupMetadata, error) {
	backupDir := "./data/backups"

	// 检查目录是否存在
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("backup directory not found")
	}

	// 列出所有备份文件
	files, err := filepath.Glob(filepath.Join(backupDir, "backup-*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to list backups: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no backup files found")
	}

	// 按修改时间排序，获取最新的
	sort.Slice(files, func(i, j int) bool {
		infoI, _ := os.Stat(files[i])
		infoJ, _ := os.Stat(files[j])
		return infoI.ModTime().After(infoJ.ModTime())
	})

	latestFile := files[0]
	info, err := os.Stat(latestFile)
	if err != nil {
		return nil, fmt.Errorf("failed to stat backup file: %w", err)
	}

	// 读取文件并计算hash
	data, err := os.ReadFile(latestFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup file: %w", err)
	}

	hash := sha256.Sum256(data)

	// 解析备份内容以获取元数据
	var backupData map[string]interface{}
	if err := json.Unmarshal(data, &backupData); err != nil {
		return nil, fmt.Errorf("failed to parse backup: %w", err)
	}

	// 提取元数据
	version := int64(0)
	recordCount := int64(0)
	lastUpdatedAt := info.ModTime()

	if metadata, ok := backupData["metadata"].(map[string]interface{}); ok {
		if v, ok := metadata["version"].(float64); ok {
			version = int64(v)
		}
		if rc, ok := metadata["record_count"].(float64); ok {
			recordCount = int64(rc)
		}
		if luat, ok := metadata["last_updated_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, luat); err == nil {
				lastUpdatedAt = t
			}
		}
	}

	// 如果没有元数据，尝试从算法数量估算
	if recordCount == 0 {
		if algorithms, ok := backupData["algorithms"].([]interface{}); ok {
			recordCount = int64(len(algorithms))
		}
	}

	return &BackupMetadata{
		Timestamp:     info.ModTime(),
		Hash:          hex.EncodeToString(hash[:]),
		Source:        "local",
		Path:          latestFile,
		Version:       version,
		RecordCount:   recordCount,
		LastUpdatedAt: lastUpdatedAt,
	}, nil
}

// restoreFromBackup 从备份恢复数据（带事务和完整性验证）
func (m *SQLiteBackupManager) restoreFromBackup(ctx context.Context, metadata *BackupMetadata) error {
	startTime := time.Now()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("🔄 Starting database restore from %s backup\n", metadata.Source)
	fmt.Printf("   Backup time: %s\n", metadata.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("   Backup hash: %s\n", metadata.Hash[:16])
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	var backupData map[string]interface{}

	// Step 1: 加载备份数据
	fmt.Print("📥 [1/5] Loading backup data... ")
	loadStart := time.Now()

	if metadata.Source == "minio" {
		// 从MinIO恢复
		obj, err := m.minio.GetObject(ctx, m.bucketName, metadata.Path, minio.GetObjectOptions{})
		if err != nil {
			fmt.Println("❌ FAILED")
			return fmt.Errorf("failed to get MinIO backup: %w", err)
		}
		defer obj.Close()

		if err := json.NewDecoder(obj).Decode(&backupData); err != nil {
			fmt.Println("❌ FAILED")
			return fmt.Errorf("failed to decode MinIO backup: %w", err)
		}
	} else {
		// 从本地恢复
		data, err := os.ReadFile(metadata.Path)
		if err != nil {
			fmt.Println("❌ FAILED")
			return fmt.Errorf("failed to read local backup: %w", err)
		}

		if err := json.Unmarshal(data, &backupData); err != nil {
			fmt.Println("❌ FAILED")
			return fmt.Errorf("failed to decode local backup: %w", err)
		}
	}
	fmt.Printf("✅ (%.2fs)\n", time.Since(loadStart).Seconds())

	// Step 2: 验证备份完整性
	fmt.Print("🔍 [2/5] Validating backup integrity... ")
	validateStart := time.Now()

	algorithmCount := 0
	if algorithms, ok := backupData["algorithms"].([]interface{}); ok {
		algorithmCount = len(algorithms)
	}

	presetDataCount := 0
	if presetData, ok := backupData["preset_data"].([]interface{}); ok {
		presetDataCount = len(presetData)
	}

	if algorithmCount == 0 && presetDataCount == 0 {
		fmt.Println("⚠️  WARNING: Backup is empty")
	} else {
		fmt.Printf("✅ (%.2fs)\n", time.Since(validateStart).Seconds())
		fmt.Printf("   Found: %d algorithms, %d preset data\n", algorithmCount, presetDataCount)
	}

	// Step 3: 开始事务恢复（确保原子性）
	fmt.Print("🔒 [3/5] Starting transactional restore... ")
	txStart := time.Now()

	tx := m.db.Begin()
	if tx.Error != nil {
		fmt.Println("❌ FAILED")
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	// 使用defer确保出错时回滚
	var restoreErr error
	defer func() {
		if restoreErr != nil {
			fmt.Print("🔙 Rolling back transaction... ")
			tx.Rollback()
			fmt.Println("✅")
		}
	}()

	fmt.Printf("✅ (%.2fs)\n", time.Since(txStart).Seconds())

	// Step 4: 清空现有数据并恢复
	fmt.Print("🗑️  [4/5] Clearing existing data... ")
	clearStart := time.Now()

	if err := tx.Exec("DELETE FROM algorithms").Error; err != nil {
		fmt.Println("❌ FAILED")
		restoreErr = fmt.Errorf("failed to clear algorithms: %w", err)
		return restoreErr
	}
	if err := tx.Exec("DELETE FROM preset_data").Error; err != nil {
		fmt.Println("❌ FAILED")
		restoreErr = fmt.Errorf("failed to clear preset data: %w", err)
		return restoreErr
	}
	fmt.Printf("✅ (%.2fs)\n", time.Since(clearStart).Seconds())

	// 恢复算法数据（带进度）
	fmt.Printf("📝 [5/5] Restoring data:\n")
	restoreStart := time.Now()

	restoredAlgorithms := 0
	failedAlgorithms := 0
	if algorithms, ok := backupData["algorithms"].([]interface{}); ok {
		totalAlgorithms := len(algorithms)
		lastProgress := 0

		for i, alg := range algorithms {
			if algMap, ok := alg.(map[string]interface{}); ok {
				var algorithm models.Algorithm
				algorithmData, _ := json.Marshal(algMap)
				json.Unmarshal(algorithmData, &algorithm)

				if result := tx.Create(&algorithm); result.Error != nil {
					fmt.Printf("   ⚠️  Algorithm %s failed: %v\n", algorithm.ID, result.Error)
					failedAlgorithms++
				} else {
					restoredAlgorithms++
				}

				// 显示进度（每10%或最后一条）
				progress := (i + 1) * 100 / totalAlgorithms
				if progress >= lastProgress+10 || i == totalAlgorithms-1 {
					fmt.Printf("   Algorithms: %d/%d (%d%%)\n", i+1, totalAlgorithms, progress)
					lastProgress = progress
				}
			}
		}
	}

	// 恢复预设数据
	restoredPresetData := 0
	failedPresetData := 0
	if presetData, ok := backupData["preset_data"].([]interface{}); ok {
		totalPresetData := len(presetData)
		for i, data := range presetData {
			if dataMap, ok := data.(map[string]interface{}); ok {
				var presetData models.PresetData
				dataData, _ := json.Marshal(dataMap)
				json.Unmarshal(dataData, &presetData)

				if result := tx.Create(&presetData); result.Error != nil {
					fmt.Printf("   ⚠️  PresetData %s failed: %v\n", presetData.ID, result.Error)
					failedPresetData++
				} else {
					restoredPresetData++
				}
			}

			// 显示进度
			if (i+1)%100 == 0 || i == totalPresetData-1 {
				fmt.Printf("   Preset data: %d/%d\n", i+1, totalPresetData)
			}
		}
	}

	fmt.Printf("   ✅ Restore completed (%.2fs)\n", time.Since(restoreStart).Seconds())

	// Step 5: 提交事务
	fmt.Print("💾 Committing transaction... ")
	commitStart := time.Now()

	if err := tx.Commit().Error; err != nil {
		fmt.Println("❌ FAILED")
		restoreErr = fmt.Errorf("failed to commit transaction: %w", err)
		return restoreErr
	}
	fmt.Printf("✅ (%.2fs)\n", time.Since(commitStart).Seconds())

	// Step 6: 验证恢复结果
	fmt.Print("🔍 Verifying restored data... ")
	verifyStart := time.Now()

	var finalAlgCount, finalPresetCount int64
	if err := m.db.Model(&models.Algorithm{}).Count(&finalAlgCount).Error; err != nil {
		fmt.Printf("⚠️  Warning: failed to verify: %v\n", err)
	} else if err := m.db.Model(&models.PresetData{}).Count(&finalPresetCount).Error; err != nil {
		fmt.Printf("⚠️  Warning: failed to verify: %v\n", err)
	} else {
		fmt.Printf("✅ (%.2fs)\n", time.Since(verifyStart).Seconds())
		fmt.Printf("   Verified: %d algorithms, %d preset data in database\n", finalAlgCount, finalPresetCount)
	}

	// 最终报告
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 Restore Summary:")
	fmt.Printf("   ✅ Algorithms: %d restored", restoredAlgorithms)
	if failedAlgorithms > 0 {
		fmt.Printf(", ⚠️  %d failed", failedAlgorithms)
	}
	fmt.Println()
	fmt.Printf("   ✅ Preset Data: %d restored", restoredPresetData)
	if failedPresetData > 0 {
		fmt.Printf(", ⚠️  %d failed", failedPresetData)
	}
	fmt.Println()
	fmt.Printf("   ⏱️  Total time: %.2fs\n", time.Since(startTime).Seconds())

	// 如果有失败项，警告但不中断启动
	if failedAlgorithms > 0 || failedPresetData > 0 {
		fmt.Println("   ⚠️  WARNING: Some items failed to restore")
		fmt.Println("   ℹ️  Service will continue with successfully restored data")
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 更新数据库元数据为备份的版本
	if err := m.restoreMetadataFromBackup(metadata); err != nil {
		fmt.Printf("Warning: failed to update database metadata: %v\n", err)
	}

	return nil
}

// restoreMetadataFromBackup 从备份恢复元数据
func (m *SQLiteBackupManager) restoreMetadataFromBackup(backupMeta *BackupMetadata) error {
	newMeta := models.DatabaseMetadata{
		Version:       backupMeta.Version,
		LastUpdatedAt: backupMeta.LastUpdatedAt,
		UpdatedBy:     "backup_restore",
		CheckpointAt:  time.Now(),
		RecordCount:   backupMeta.RecordCount,
	}

	return m.db.Create(&newMeta).Error
}

// BackupToMinIO 备份数据到 MinIO（优先）或本地（fallback）
func (m *SQLiteBackupManager) BackupToMinIO() error {
	ctx := context.Background()

	// 获取当前数据库元数据
	meta, err := m.getDatabaseMetadata()
	if err != nil {
		return fmt.Errorf("failed to get database metadata: %w", err)
	}

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

	// 包含元数据的备份
	backupData := map[string]interface{}{
		"algorithms":  algorithms,
		"versions":    versions,
		"preset_data": presetData,
		"jobs":        jobs,
		"backuped_at": time.Now(),
		"backup_type": "sqlite",
		"metadata": map[string]interface{}{
			"version":         meta.Version,
			"record_count":    meta.RecordCount,
			"last_updated_at": meta.LastUpdatedAt,
		},
	}

	backupJSON, err := json.MarshalIndent(backupData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal backup data: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")

	// 优先备份到 MinIO
	minioSuccess := false
	if err := m.backupJSONToMinIO(ctx, backupJSON, timestamp); err != nil {
		fmt.Printf("Warning: MinIO JSON backup failed, falling back to local: %v\n", err)
	} else {
		minioSuccess = true
		fmt.Printf("JSON backup saved to MinIO: backup-%s.json (version: %d)\n", timestamp, meta.Version)
	}

	// MinIO 失败时才备份到本地
	if !minioSuccess {
		if err := m.saveLocalBackup(backupJSON, timestamp); err != nil {
			return fmt.Errorf("both MinIO and local JSON backup failed: %w", err)
		}
		fmt.Printf("JSON backup saved to local (fallback): backup-%s.json (version: %d)\n", timestamp, meta.Version)
	}

	// 备份数据库文件（同样优先 MinIO）
	dbSuccess := false
	if err := m.backupDBFileToMinIO(timestamp); err != nil {
		fmt.Printf("Warning: MinIO database file backup failed, falling back to local: %v\n", err)
	} else {
		dbSuccess = true
		fmt.Printf("Database file backed up to MinIO: db-backup-%s.db\n", timestamp)
	}

	// MinIO 失败时才备份数据库文件到本地
	if !dbSuccess {
		if err := m.saveLocalDBBackup(timestamp); err != nil {
			fmt.Printf("Warning: local database file backup also failed: %v\n", err)
		} else {
			fmt.Printf("Database file backed up to local (fallback): db-backup-%s.db\n", timestamp)
		}
	}

	// 异步清理旧备份
	go m.cleanupOldBackups()

	return nil
}

// backupJSONToMinIO 将 JSON 备份上传到 MinIO
func (m *SQLiteBackupManager) backupJSONToMinIO(ctx context.Context, backupJSON []byte, timestamp string) error {
	// 上传带时间戳的备份
	backupPath := fmt.Sprintf("database-backup/backup-%s.json", timestamp)
	_, err := m.minio.PutObject(ctx, m.bucketName, backupPath,
		bytes.NewReader(backupJSON), int64(len(backupJSON)),
		minio.PutObjectOptions{
			ContentType: "application/json",
		})
	if err != nil {
		return fmt.Errorf("failed to upload backup to MinIO: %w", err)
	}

	// 更新 latest 备份
	latestPath := "database-backup/latest.json"
	_, err = m.minio.PutObject(ctx, m.bucketName, latestPath,
		bytes.NewReader(backupJSON), int64(len(backupJSON)),
		minio.PutObjectOptions{
			ContentType: "application/json",
		})
	if err != nil {
		return fmt.Errorf("failed to update latest backup: %w", err)
	}

	return nil
}

// saveLocalBackup 保存本地 JSON 备份
func (m *SQLiteBackupManager) saveLocalBackup(data []byte, timestamp string) error {
	backupDir := "./data/backups"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	backupFile := filepath.Join(backupDir, fmt.Sprintf("backup-%s.json", timestamp))
	if err := os.WriteFile(backupFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	return nil
}

// saveLocalDBBackup 保存本地数据库文件备份
func (m *SQLiteBackupManager) saveLocalDBBackup(timestamp string) error {
	backupDir := "./data/backups"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// 读取数据库文件
	data, err := os.ReadFile(m.dbPath)
	if err != nil {
		return fmt.Errorf("failed to read database file: %w", err)
	}

	// 保存到本地
	backupFile := filepath.Join(backupDir, fmt.Sprintf("db-backup-%s.db", timestamp))
	if err := os.WriteFile(backupFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write database backup file: %w", err)
	}

	return nil
}

// cleanupOldBackups 清理旧备份（MinIO 和本地）
func (m *SQLiteBackupManager) cleanupOldBackups() {
	// ctx := context.Background()

	// // 清理 MinIO 旧的 JSON 备份（保留最近 10 个）
	// jsonBackups := m.listBackupsByPrefix(ctx, "database-backup/backup-")
	// if len(jsonBackups) > 10 {
	// 	for _, key := range jsonBackups[:len(jsonBackups)-10] {
	// 		if err := m.minio.RemoveObject(ctx, m.bucketName, key, minio.RemoveObjectOptions{}); err != nil {
	// 			fmt.Printf("Failed to delete old MinIO JSON backup %s: %v\n", key, err)
	// 		} else {
	// 			fmt.Printf("Deleted old MinIO JSON backup: %s\n", key)
	// 		}
	// 	}
	// }

	// // 清理 MinIO 旧的数据库文件备份（保留最近 5 个）
	// dbBackups := m.listBackupsByPrefix(ctx, "database-backup/db-backup-")
	// if len(dbBackups) > 5 {
	// 	for _, key := range dbBackups[:len(dbBackups)-5] {
	// 		if err := m.minio.RemoveObject(ctx, m.bucketName, key, minio.RemoveObjectOptions{}); err != nil {
	// 			fmt.Printf("Failed to delete old MinIO DB backup %s: %v\n", key, err)
	// 		} else {
	// 			fmt.Printf("Deleted old MinIO DB backup: %s\n", key)
	// 		}
	// 	}
	// }

	// 清理本地旧备份
	m.cleanupLocalBackups()
}

// listBackupsByPrefix 列出指定前缀的备份文件
func (m *SQLiteBackupManager) listBackupsByPrefix(ctx context.Context, prefix string) []string {
	objectCh := m.minio.ListObjects(ctx, m.bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	var backups []string
	for object := range objectCh {
		if object.Err != nil {
			fmt.Printf("Error listing backups: %v\n", object.Err)
			return backups
		}
		// 排除 latest 文件
		if object.Key != "database-backup/latest.json" && object.Key != "database-backup/latest.db" && object.Key != "database-backup/final-backup.db" {
			backups = append(backups, object.Key)
		}
	}

	sort.Strings(backups)
	return backups
}

// cleanupLocalBackups 清理本地旧备份（JSON 和数据库文件）
func (m *SQLiteBackupManager) cleanupLocalBackups() {
	backupDir := "./data/backups"

	// 清理 JSON 备份（保留最近 5 个）
	jsonFiles, err := filepath.Glob(filepath.Join(backupDir, "backup-*.json"))
	if err == nil {
		sort.Strings(jsonFiles)
		if len(jsonFiles) > 5 {
			for _, file := range jsonFiles[:len(jsonFiles)-5] {
				if err := os.Remove(file); err != nil {
					fmt.Printf("Failed to delete local JSON backup %s: %v\n", file, err)
				} else {
					fmt.Printf("Deleted old local JSON backup: %s\n", file)
				}
			}
		}
	}

	// 清理数据库文件备份（保留最近 3 个）
	dbFiles, err := filepath.Glob(filepath.Join(backupDir, "db-backup-*.db"))
	if err == nil {
		sort.Strings(dbFiles)
		if len(dbFiles) > 3 {
			for _, file := range dbFiles[:len(dbFiles)-3] {
				if err := os.Remove(file); err != nil {
					fmt.Printf("Failed to delete local DB backup %s: %v\n", file, err)
				} else {
					fmt.Printf("Deleted old local DB backup: %s\n", file)
				}
			}
		}
	}
}

// StartBackupScheduler 启动备份调度器
func (m *SQLiteBackupManager) StartBackupScheduler() error {
	ticker := time.NewTicker(m.backupInterval)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-m.stopBackup:
				return
			case <-ticker.C:
				if err := m.BackupToMinIO(); err != nil {
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

// backupDBFileToMinIO 备份数据库文件到 MinIO
func (m *SQLiteBackupManager) backupDBFileToMinIO(timestamp string) error {
	ctx := context.Background()

	// 读取数据库文件
	dbFile, err := os.Open(m.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database file: %w", err)
	}
	defer dbFile.Close()

	// 获取文件大小
	fileInfo, err := dbFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat database file: %w", err)
	}

	// 上传到 MinIO（带时间戳）
	dbBackupPath := fmt.Sprintf("database-backup/db-backup-%s.db", timestamp)
	_, err = m.minio.PutObject(ctx, m.bucketName, dbBackupPath,
		dbFile, fileInfo.Size(),
		minio.PutObjectOptions{
			ContentType: "application/octet-stream",
		})
	if err != nil {
		return fmt.Errorf("failed to upload database file to MinIO: %w", err)
	}

	// 重新打开文件用于 latest 备份
	dbFile.Seek(0, 0)

	// 更新 latest 数据库文件
	latestDBPath := "database-backup/latest.db"
	_, err = m.minio.PutObject(ctx, m.bucketName, latestDBPath,
		dbFile, fileInfo.Size(),
		minio.PutObjectOptions{
			ContentType: "application/octet-stream",
		})
	if err != nil {
		return fmt.Errorf("failed to update latest database file: %w", err)
	}

	// 只打印简短的文件名，避免重复
	return nil
}

// BackupDBFile 手动备份数据库文件到 MinIO（给 sqlite.go 调用）
func (m *SQLiteBackupManager) BackupDBFile(destPath string) error {
	// 删除已存在的备份文件（如果存在）
	if _, err := os.Stat(destPath); err == nil {
		if err := os.Remove(destPath); err != nil {
			return fmt.Errorf("failed to remove existing backup file: %w", err)
		}
	}

	// 执行 VACUUM INTO 创建本地备份
	sqlDB, err := m.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	query := fmt.Sprintf("VACUUM INTO '%s'", destPath)
	if _, err := sqlDB.Exec(query); err != nil {
		return fmt.Errorf("VACUUM INTO failed: %w", err)
	}

	// 上传到 MinIO
	ctx := context.Background()
	dbFile, err := os.Open(destPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer dbFile.Close()

	fileInfo, err := dbFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat backup file: %w", err)
	}

	backupPath := "database-backup/final-backup.db"
	_, err = m.minio.PutObject(ctx, m.bucketName, backupPath,
		dbFile, fileInfo.Size(),
		minio.PutObjectOptions{
			ContentType: "application/octet-stream",
		})
	if err != nil {
		return fmt.Errorf("failed to upload final backup to MinIO: %w", err)
	}

	fmt.Printf("Final database backup uploaded to MinIO: %s\n", backupPath)
	return nil
}

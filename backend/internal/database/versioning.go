package database

import (
	"algorithm-platform/internal/models"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// VersioningPlugin GORM插件，用于自动更新数据库版本号
type VersioningPlugin struct {
	db *gorm.DB
}

// Name 插件名称
func (p *VersioningPlugin) Name() string {
	return "VersioningPlugin"
}

// Initialize 初始化插件
func (p *VersioningPlugin) Initialize(db *gorm.DB) error {
	p.db = db
	
	// 注册回调：在创建、更新、删除后更新版本号
	if err := db.Callback().Create().After("gorm:after_create").Register("versioning:after_create", p.afterWrite); err != nil {
		return err
	}
	if err := db.Callback().Update().After("gorm:after_update").Register("versioning:after_update", p.afterWrite); err != nil {
		return err
	}
	if err := db.Callback().Delete().After("gorm:after_delete").Register("versioning:after_delete", p.afterWrite); err != nil {
		return err
	}
	
	return nil
}

// afterWrite 写操作后的回调
func (p *VersioningPlugin) afterWrite(db *gorm.DB) {
	// 只在主要表变更时更新版本
	tableName := db.Statement.Table
	if tableName == "algorithms" || tableName == "preset_data" || tableName == "versions" {
		// 异步更新版本号，避免影响主操作性能
		go p.incrementVersion()
	}
}

// incrementVersion 递增数据库版本号
func (p *VersioningPlugin) incrementVersion() {
	// 获取当前最大版本号
	var currentMeta models.DatabaseMetadata
	p.db.Order("version DESC").First(&currentMeta)

	// 统计记录数
	var count int64
	p.db.Model(&models.Algorithm{}).Count(&count)

	newMeta := models.DatabaseMetadata{
		Version:       currentMeta.Version + 1,
		LastUpdatedAt: time.Now(),
		UpdatedBy:     "auto",
		CheckpointAt:  time.Now(),
		RecordCount:   count,
	}

	if err := p.db.Create(&newMeta).Error; err != nil {
		fmt.Printf("Warning: failed to update database version: %v\n", err)
	}
}

// InstallVersioning 安装版本控制插件
func InstallVersioning(db *gorm.DB) error {
	plugin := &VersioningPlugin{}
	return db.Use(plugin)
}

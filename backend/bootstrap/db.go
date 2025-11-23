package bootstrap

import (
	"github.com/dbehnke/allstar-nexus/backend/config"
	"github.com/dbehnke/allstar-nexus/backend/models"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite" // Register sqlite driver
)

// InitDB initializes the GORM database connection and runs migrations
func InitDB(cfg *config.Config, logger *zap.Logger) (*gorm.DB, error) {
	// Initialize GORM database with modernc.org/sqlite (pure Go, no CGO)
	gormDB, err := gorm.Open(sqlite.New(sqlite.Config{
		DriverName: "sqlite",
		DSN:        cfg.DBPath,
	}), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Set PRAGMA settings for optimized write performance
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, err
	}
	if _, err := sqlDB.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		logger.Warn("failed to set journal_mode=WAL", zap.Error(err))
	}
	if _, err := sqlDB.Exec("PRAGMA synchronous=NORMAL;"); err != nil {
		logger.Warn("failed to set synchronous=NORMAL", zap.Error(err))
	}

	// Auto-migrate models
	if err := gormDB.AutoMigrate(
		&models.User{},
		&models.TransmissionLog{},
		&models.NodeInfo{},
		&models.LinkStat{},
		&models.CallsignProfile{},
		&models.LevelConfig{},
		&models.XPActivityLog{},
		&models.TallyState{},
	); err != nil {
		return nil, err
	}

	logger.Info("GORM database initialized successfully")
	return gormDB, nil
}

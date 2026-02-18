package config

import (
	"fmt"
	"net/url"
	"time"

	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func ConnectDatabase(c DBConfig) (*gorm.DB, error) {
	// URL-encode user and password so special characters (e.g. @ in password) don't break the DSN
	userInfo := url.UserPassword(c.User, c.Password)
	dsn := fmt.Sprintf("sqlserver://%s@%s?database=%s&encrypt=%t&trustServerCertificate=%t",
		userInfo.String(), c.Server, c.Database, c.Encrypt, c.TrustServerCertificate)

	db, err := gorm.Open(sqlserver.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(c.PoolMin)
	sqlDB.SetMaxOpenConns(c.PoolMax)
	sqlDB.SetConnMaxLifetime(time.Duration(c.IdleTimeout) * time.Millisecond)

	return db, nil
}

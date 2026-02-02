package config

import (
	"fmt"
	"time"

	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func ConnectDatabase(c DBConfig) (*gorm.DB, error) {

	// dsn := "sqlserver://user:password@server:port?database=dbname"
	dsn := fmt.Sprintf("sqlserver://%s:%s@%s?database=%s&encrypt=%t&trustServerCertificate=%t",
		c.User, c.Password, c.Server, c.Database, c.Encrypt, c.TrustServerCertificate)

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

package config

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func ConnectDatabase() {
	c := AppConfig.DB
	
	// dsn := "sqlserver://user:password@server:port?database=dbname"
	dsn := fmt.Sprintf("sqlserver://%s:%s@%s?database=%s&encrypt=%t&trustServerCertificate=%t",
		c.User, c.Password, c.Server, c.Database, c.Encrypt, c.TrustServerCertificate)

	db, err := gorm.Open(sqlserver.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database instance: %v", err)
	}

	sqlDB.SetMaxIdleConns(c.PoolMin)
	sqlDB.SetMaxOpenConns(c.PoolMax)
	sqlDB.SetConnMaxLifetime(time.Duration(c.IdleTimeout) * time.Millisecond)

	DB = db
	log.Println("🗄️ Database connected successfully")
}

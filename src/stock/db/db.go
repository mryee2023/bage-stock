package db

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"vps-stock/src/stock/vars"

	"gorm.io/gorm"
)

var once sync.Once
var db *gorm.DB

func Open(cfg *vars.Config) {
	once.Do(func() {
		var err error
		db, err = gorm.Open(sqlite.Open(cfg.Db), &gorm.Config{
			QueryFields:            true,
			SkipDefaultTransaction: true,
			NowFunc: func() time.Time {
				return time.Now().Local()
			},
		})
		if err != nil {
			log.Fatalf("open %v error: %v", cfg.Db, err)
		}
		db.AutoMigrate(&Kind{})
		db = db.Debug()
		sqlDb, err := db.DB()
		if err != nil {
			log.Fatalf("get db error: %v", err)
		}
		sqlDb.SetMaxIdleConns(100)
		sqlDb.SetMaxOpenConns(100)
		sqlDb.SetConnMaxLifetime(time.Hour)
	})

}

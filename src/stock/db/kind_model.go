package db

import (
	"time"
)

type Kind struct {
	ID        int       `gorm:"primaryKey,autoIncrement"`
	Kind      string    `gorm:"unique"`
	Stock     int       `gorm:"default:0"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func AddOrUpdateKind(k *Kind) error {
	var exists *Kind
	v := db.Model(&Kind{}).Where("kind = ?", k.Kind).Limit(1).Find(&exists)
	if v.Error != nil {
		return v.Error
	}
	if v.RowsAffected == 0 {
		return db.Create(k).Error
	}
	return db.Model(&Kind{}).Where("kind = ?", k.Kind).Updates(k).Error
}

func GetKindByKind(kind string) (*Kind, error) {
	var k Kind
	v := db.Model(&Kind{}).Where("kind = ?", kind).Limit(1).Find(&k)
	if v.Error != nil {
		return nil, v.Error
	}
	if v.RowsAffected == 0 {
		return nil, nil
	}
	return &k, nil
}

func GetKinds() ([]*Kind, error) {
	var kinds []*Kind
	v := db.Model(&Kind{}).Find(&kinds)
	if v.Error != nil {
		return nil, v.Error
	}
	return kinds, nil
}

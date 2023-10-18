package storage

import (
	"github.com/F1997/nightingale/pkg/ormx"

	"gorm.io/gorm"
)

func New(cfg ormx.DBConfig) (*gorm.DB, error) {
	// 调用 ormx.New(cfg) 来创建一个 gorm.DB 数据库连接对象
	db, err := ormx.New(cfg)
	if err != nil {
		return nil, err
	}

	return db, nil
}

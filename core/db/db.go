package db

import (
	"apipig/core/api"
	"apipig/core/api/request"
	"apipig/core/api/response"
	"apipig/global"
	"apipig/middleware"
	"apipig/toolkit"
	"apipig/toolkit/snowflake"
	"database/sql"
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var NotFoundError = errors.New("not found")

func GetId() snowflake.ID {
	return toolkit.Id(global.CONFIG.System.Node)
}

func NewModel(c *fiber.Ctx) api.MODEL {
	tc := middleware.GetTokenClaims(c)
	return api.MODEL{
		ID:        GetId(),
		CreatedId: tc.ID,
		CreatedBy: tc.Username,
		CreatedAt: toolkit.GetNowUnixMilli(),
	}
}

func GetDB(value interface{}) *gorm.DB {
	return global.DB.Model(value)
}

func Transaction(fc func(tx *gorm.DB) error, opts ...*sql.TxOptions) error {
	return global.DB.Transaction(fc, opts...)
}

func Create(value interface{}) (bool, error) {
	err := global.DB.Create(value).Error
	return err == nil, err
}

func DeleteByIds(value interface{}, ids interface{}) (bool, error) {
	err := global.DB.Where("id IN ?", ids).Delete(&value).Error
	return err == nil, err
}

func Update(values interface{}) (bool, error) {
	err := global.DB.Updates(values).Error
	return err == nil, err
}

func CountById(value interface{}, id snowflake.ID) (count int64) {
	GetDB(value).Where("id = ?", id).Count(&count)
	return
}

func ExistInPid(value interface{}, ids []snowflake.ID) bool {
	var count int64
	if err := GetDB(value).Where("pid IN ?", ids).Count(&count).Error; err != nil {
		return true
	}
	return count > 0
}

func Exist(value interface{}, where func(t *gorm.DB) *gorm.DB) bool {
	var count int64
	if err := where(GetDB(value)).Count(&count).Error; err != nil {
		return true
	}
	return count > 0
}

func GetById(dest interface{}, id snowflake.ID) error {
	return global.DB.First(dest, id).Error
}

func GetByIds(value interface{}, ids []snowflake.ID) error {
	return global.DB.Where("id IN ?", ids).Find(value).Error
}

func Page(db *gorm.DB, pageInfo request.PageInfo, records interface{}) (result response.PageResult, err error) {
	return CallPage(db, pageInfo, func(db *gorm.DB, t *int64) error {
		return db.Count(t).Error
	}, func(db *gorm.DB, l int, s int) (interface{}, error) {
		r := db.Limit(l).Offset(s).Find(&records).Error
		return records, r
	})
}

func PageRaw(pageInfo request.PageInfo, records interface{}, sqlCount string,
	sqlRecords string, args []interface{}) (result response.PageResult, err error) {
	return CallPage(global.DB, pageInfo, func(db *gorm.DB, t *int64) error {
		if len(args) == 0 {
			return db.Raw(sqlCount).Scan(t).Error
		}
		return db.Raw(sqlCount, args).Scan(t).Error
	}, func(db *gorm.DB, l int, s int) (interface{}, error) {
		_sql := fmt.Sprintf("%s LIMIT %d OFFSET %d", sqlRecords, l, s)
		if len(args) == 0 {
			r := db.Raw(_sql).Scan(&records).Error
			return records, r
		}

		r := db.Raw(_sql, args).Scan(&records).Error
		return records, r
	})
}

func CallPage(db *gorm.DB, pageInfo request.PageInfo, callCount func(db *gorm.DB, t *int64) error,
	callRecords func(db *gorm.DB, l int, s int) (records interface{}, err error)) (result response.PageResult, err error) {
	var init = true
	var flag = true
	page, pageSize, offset := pageInfo.PageOffset()
	var total int64
	if pageInfo.SearchCount == 0 {
		// 查询总数
		err = callCount(db, &total)
		if err != nil || total == 0 {
			flag = false
		}
	}

	var records interface{}
	if flag {
		// 分页查询
		records, err = callRecords(db, pageSize, offset)
		if err == nil {
			init = false
		}
	}
	if init {
		// 默认返回空数组
		records = make([]interface{}, 0)
	}
	result = response.PageResult{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Records:  records,
	}
	return
}

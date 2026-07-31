package service

import (
	"fmt"
	"testing"

	"apipig/app/ai/model"
	aiReq "apipig/app/ai/model/request"
	coreAPI "apipig/core/api"
	coreReq "apipig/core/api/request"
	"apipig/toolkit/snowflake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCallLogPageReturnsTotalAndRequestedPage(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:call-log-page?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&model.CallLog{}))

	logs := make([]model.CallLog, 25)
	for index := range logs {
		logs[index] = model.CallLog{
			MODEL:     coreAPI.MODEL{ID: snowflake.ID(index + 1), CreatedAt: int64(index + 1)},
			RequestID: fmt.Sprintf("request-%02d", index+1),
			Model:     "gpt-test",
			Path:      "/v1/chat/completions",
			Method:    "POST",
			Success:   1,
		}
	}
	require.NoError(t, database.Create(&logs).Error)

	logService := &CallLogService{store: newGormAIStore(func() *gorm.DB { return database })}
	result, err := logService.Page(&aiReq.CallLogPageParams{PageInfo: coreReq.PageInfo{Page: 2, PageSize: 10}})
	require.NoError(t, err)
	assert.EqualValues(t, 25, result.Total)
	assert.Equal(t, 2, result.Page)
	assert.Equal(t, 10, result.PageSize)
	records, ok := result.Records.([]model.CallLog)
	require.True(t, ok)
	assert.Len(t, records, 10)
}

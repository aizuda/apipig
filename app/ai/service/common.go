package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"apipig/core/api/request"
)

const (
	maxAIPage     = 1_000_000
	maxAIPageSize = 1_000
	maxAIBulkIDs  = 1_000
)

// pageParams 约束所有包含通用分页参数的 AI 查询请求。
type pageParams interface {
	GetPageInfo() request.PageInfo
}

func validateAIBulkIDs(idsReq *request.IdsReq, emptyMessage string) error {
	if idsReq == nil || len(idsReq.Ids) == 0 {
		return errors.New(emptyMessage)
	}
	if len(idsReq.Ids) > maxAIBulkIDs {
		return fmt.Errorf("单次批量操作不能超过 %d 条", maxAIBulkIDs)
	}
	for _, id := range idsReq.Ids {
		if id == 0 {
			return errors.New("批量操作包含无效 ID")
		}
	}
	return nil
}

// pageInfo 获取分页参数；参数为空时返回统一默认值。
func pageInfo[T pageParams](params T) request.PageInfo {
	if any(params) == nil {
		return normalizeAIPageInfo(request.PageInfo{})
	}
	return normalizeAIPageInfo(params.GetPageInfo())
}

// normalizeAIPageInfo 统一分页边界，避免异常大页码造成整数溢出或超大数据库查询。
func normalizeAIPageInfo(info request.PageInfo) request.PageInfo {
	if info.Page <= 0 {
		info.Page = 1
	} else if info.Page > maxAIPage {
		info.Page = maxAIPage
	}
	if info.PageSize <= 0 {
		info.PageSize = 10
	} else if info.PageSize > maxAIPageSize {
		info.PageSize = maxAIPageSize
	}
	if info.SearchCount != 1 {
		info.SearchCount = 0
	}
	return info
}

// defaultString 在字段为空时返回指定默认值。
func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

// decodeStrictJSON 拒绝未知字段和尾随内容，避免配置项拼写错误后被静默忽略。
func decodeStrictJSON(value string, destination any) error {
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return io.ErrUnexpectedEOF
		}
		return err
	}
	return nil
}

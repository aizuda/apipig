package service

import "apipig/core/api/request"

// pageParams 约束所有包含通用分页参数的 AI 查询请求。
type pageParams interface {
	GetPageInfo() request.PageInfo
}

// pageInfo 获取分页参数；参数为空时返回统一默认值。
func pageInfo[T pageParams](params T) request.PageInfo {
	if any(params) == nil {
		return request.PageInfo{Page: 1, PageSize: 10}
	}
	return params.GetPageInfo()
}

// defaultString 在字段为空时返回指定默认值。
func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

package service

import coreReq "apipig/core/api/request"

type pageParams interface{ GetPageInfo() coreReq.PageInfo }

func pageInfo[T pageParams](params T) coreReq.PageInfo {
	if any(params) == nil {
		return coreReq.PageInfo{Page: 1, PageSize: 10}
	}
	return params.GetPageInfo()
}

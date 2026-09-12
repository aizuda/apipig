package request

import "apipig/toolkit/snowflake"

const (
	DefaultPage     = 1
	DefaultPageSize = 10
	MaxPage         = 1_000_000
	MaxPageSize     = 1_000
)

// PageInfo Paging common input parameter structure
type PageInfo struct {
	SearchCount int `json:"searchCount,omitempty"` // 查询总数 0，是 1，否
	Page        int `json:"page,omitempty"`        // 页码
	PageSize    int `json:"pageSize,omitempty"`    // 每页大小
}

func (p *PageInfo) PageOffset() (int, int, int) {
	if p == nil {
		return DefaultPage, DefaultPageSize, 0
	}
	page := p.Page
	if page <= 0 {
		page = DefaultPage
	} else if page > MaxPage {
		page = MaxPage
	}
	pageSize := p.PageSize
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	} else if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return page, pageSize, pageSize * (page - 1)
}

// GetById Find by id structure
type GetById struct {
	ID snowflake.ID `json:"id" form:"id" swaggertype:"string"` // 主键ID
}

type IdsReq struct {
	Ids []snowflake.ID `json:"ids" form:"ids" swaggertype:"array,string"` //ID数组
}

type IdsRemarkReq struct {
	Ids    []snowflake.ID `json:"ids" form:"ids" swaggertype:"array,string"` //ID数组
	Remark string         `json:"remark" form:"remark"`                      //备注
}

// IdRelIdsReq 一对多关联
type IdRelIdsReq struct {
	RelIds []snowflake.ID `json:"relIds" form:"relIds" swaggertype:"array,string"` //关联ID数组
	ID     snowflake.ID   `json:"id" form:"id" swaggertype:"string"`               // 主键ID
}

type Empty struct{}

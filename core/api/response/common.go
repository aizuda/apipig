package response

type PageResult struct {
	Total    int64       `json:"total"`    // 总数
	Page     int         `json:"page"`     // 页码
	PageSize int         `json:"pageSize"` // 每页大小
	Records  interface{} `json:"records"`  // 数据列表
}

func EmptyPageResult(page, pageSize int) PageResult {
	return PageResult{
		Total:    0,
		Page:     page,
		PageSize: pageSize,
		Records:  make([]interface{}, 0),
	}
}

// SelectResult 选择列表响应对应对象
type SelectResult struct {
	ID   uint   `json:"id"`   // 主键ID
	Name string `json:"name"` // 名称
}

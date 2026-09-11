package response

import "apipig/app/ai/model"

// CallLogRecord 表示管理面调用日志列表记录，附带访问令牌名称。
// 令牌明文与哈希不会返回给调用方，仅展示用户自定义的令牌名称。
type CallLogRecord struct {
	model.CallLog
	AccessTokenName string `json:"accessTokenName"` // 调用方访问令牌名称，令牌不存在时为空
}

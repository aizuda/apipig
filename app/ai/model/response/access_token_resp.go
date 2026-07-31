package response

import "apipig/app/ai/model"

type AccessTokenPageRecord struct {
	model.AccessToken
	ChannelName  string                 `json:"channelName"`
	ProviderName string                 `json:"providerName"`
	Tags         []model.AccessTokenTag `json:"tags"`
}

// AccessTokenSaveResult 返回访问令牌保存结果。
// Token 仅在系统新建密钥时返回一次，后续查询和编辑不会再返回明文。
type AccessTokenSaveResult struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"`
}

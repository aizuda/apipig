package response

import "apipig/app/sys/model"

type CaptchaResp struct {
	Token       string `json:"token"`
	Base64Image string `json:"base64Image"`
}

type PublicKeyResp struct {
	UUID      string `json:"uuid"`
	PublicKey string `json:"publicKey"`
}

type LoginResp struct {
	User         model.User `json:"user"`
	Token        string     `json:"token"`
	RefreshToken string     `json:"refreshToken"`
	LoginType    string     `json:"loginType"`
}

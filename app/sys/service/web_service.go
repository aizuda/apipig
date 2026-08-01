package service

import (
	"context"
	"errors"
	"strings"
	"time"

	aiService "apipig/app/ai/service"
	"apipig/app/sys/model"
	sysReq "apipig/app/sys/model/request"
	sysResp "apipig/app/sys/model/response"
	coreAPI "apipig/core/api"
	"apipig/core/api/response"
	"apipig/global"
	"apipig/middleware"
	"apipig/toolkit"
	"apipig/toolkit/captcha"
	"apipig/toolkit/snowflake"

	"github.com/allegro/bigcache/v3"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

type WebService struct{}

const apiTokenAuthorizationDuration = 7 * 24 * time.Hour

var cache1Minute, cache1MinuteErr = bigcache.New(context.Background(), bigcache.DefaultConfig(time.Minute))
var loginPrivateKeyCache, loginPrivateKeyCacheErr = bigcache.New(context.Background(), bigcache.DefaultConfig(5*time.Minute))

func (s *WebService) Captcha(c *fiber.Ctx) error {
	if cache1MinuteErr != nil || cache1Minute == nil {
		global.LOG.Error("captcha cache init failed", zap.Error(cache1MinuteErr))
		return errors.New("captcha cache initialization failed")
	}
	data, err := captcha.New(&captcha.Options{Curve: 2, Length: 4, Width: 160, Height: 50})
	if err != nil || data == nil {
		global.LOG.Error("captcha generate failed", zap.Error(err))
		return errors.New("captcha generation failed")
	}
	token := toolkit.Uuid()
	if err = cache1Minute.Set("captcha:"+token, []byte(data.Text)); err != nil {
		return err
	}
	return response.Ok(c, sysResp.CaptchaResp{Token: token, Base64Image: data.EncodeB64string()})
}

func (s *WebService) PublicKey() (sysResp.PublicKeyResp, error) {
	if loginPrivateKeyCacheErr != nil || loginPrivateKeyCache == nil {
		global.LOG.Error("login private key cache init failed", zap.Error(loginPrivateKeyCacheErr))
		return sysResp.PublicKeyResp{}, errors.New("login private key cache initialization failed")
	}
	privateKey, publicKey, err := toolkit.GenerateRSAKeyPair(2048)
	if err != nil {
		return sysResp.PublicKeyResp{}, errors.New("failed to generate login public key")
	}
	uuid := strings.ReplaceAll(toolkit.Uuid(), "-", "")
	if err = loginPrivateKeyCache.Set(uuid, privateKey); err != nil {
		return sysResp.PublicKeyResp{}, errors.New("failed to cache login private key")
	}
	return sysResp.PublicKeyResp{UUID: uuid, PublicKey: publicKey}, nil
}

func (s *WebService) Login(c *fiber.Ctx, params *sysReq.LoginParams) (loginResp sysResp.LoginResp, err error) {
	if err = s.verifyCaptcha(params.CaptchaToken, params.CaptchaCode); err != nil {
		return loginResp, err
	}
	username, passwordText, err := s.loginDecrypt(params)
	if err != nil {
		return loginResp, err
	}
	user, err := SysService.UserService.GetByUsername(username)
	if err != nil {
		return loginResp, err
	}
	if toolkit.GetPassword(passwordText, username) != user.Password {
		return loginResp, errors.New("invalid username or password")
	}
	sid := toolkit.Uuid()
	loginResp, err = s.tokenNext(user, sid)
	if err == nil {
		_, err = SysService.UserService.CreateSession(c, loginResp.User.ID, loginResp.User.Username, sid)
	}
	return loginResp, err
}

func (s *WebService) AuthorizeToken(c *fiber.Ctx, params *sysReq.TokenAuthorizationParams) (sysResp.LoginResp, error) {
	if err := s.verifyCaptcha(params.CaptchaToken, params.CaptchaCode); err != nil {
		return sysResp.LoginResp{}, err
	}
	token, err := aiService.AiService.AccessTokenService.Authenticate(params.Token, c.IP())
	if err != nil {
		return sysResp.LoginResp{}, err
	}
	return s.apiTokenNext(token.ID, token.Name, toolkit.Uuid())
}

func (s *WebService) verifyCaptcha(token string, input string) error {
	code, err := cache1Minute.Get("captcha:" + token)
	if err != nil || !strings.EqualFold(input, string(code)) {
		return errors.New("captcha is incorrect")
	}
	return nil
}

func (s *WebService) loginDecrypt(params *sysReq.LoginParams) (string, string, error) {
	if loginPrivateKeyCacheErr != nil || loginPrivateKeyCache == nil {
		global.LOG.Error("login private key cache init failed", zap.Error(loginPrivateKeyCacheErr))
		return "", "", errors.New("login private key cache initialization failed")
	}
	if params.UUID == "" {
		return "", "", errors.New("failed to parse login private key")
	}
	privateKeyBytes, err := loginPrivateKeyCache.Get(params.UUID)
	if err != nil {
		return "", "", errors.New("login key expired; refresh and try again")
	}
	_ = loginPrivateKeyCache.Delete(params.UUID)
	privateKey, err := toolkit.ParseRSAPrivateKey(privateKeyBytes)
	if err != nil {
		return "", "", errors.New("failed to parse login private key")
	}
	username, err := toolkit.DecryptRSAOAEP(privateKey, params.Username)
	if err != nil {
		return "", "", errors.New("failed to decrypt login parameters")
	}
	password, err := toolkit.DecryptRSAOAEP(privateKey, params.Password)
	if err != nil {
		return "", "", errors.New("failed to decrypt login parameters")
	}
	return username, password, nil
}

func (s *WebService) tokenNext(user model.User, sid string) (loginResp sysResp.LoginResp, err error) {
	nowTime := toolkit.GetNowLocal()
	j := middleware.NewJWT()
	accessToken, accessErr := j.CreateToken(middleware.TokenClaims{
		SID: sid, ID: user.ID, NickName: user.NickName, Username: user.Username, LoginType: middleware.LoginTypeAccount,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(nowTime.Add(time.Duration(j.ExpiresTime) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(nowTime),
		},
	})
	refreshToken, refreshErr := j.CreateToken(middleware.RefreshTokenClaims{
		SID: sid, ID: user.ID, LoginType: middleware.LoginTypeAccount,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(nowTime.Add(time.Duration(j.ExpiresTime+5) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(nowTime),
		},
	})
	if accessErr != nil || refreshErr != nil {
		global.LOG.Error("failed to create login session", zap.Errors("errors", []error{accessErr, refreshErr}))
		return loginResp, errors.New("failed to create login session")
	}
	return sysResp.LoginResp{User: user, Token: accessToken, RefreshToken: refreshToken, LoginType: middleware.LoginTypeAccount}, nil
}

func (s *WebService) apiTokenNext(accessTokenID snowflake.ID, name string, sid string) (sysResp.LoginResp, error) {
	nowTime := toolkit.GetNowLocal()
	expiresAt := nowTime.Add(apiTokenAuthorizationDuration)
	j := middleware.NewJWT()
	accessToken, accessErr := j.CreateToken(middleware.TokenClaims{
		SID: sid, ID: accessTokenID, Username: name, NickName: name,
		LoginType: middleware.LoginTypeAPIToken, AccessTokenID: accessTokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(nowTime),
		},
	})
	if accessErr != nil {
		return sysResp.LoginResp{}, errors.New("failed to create login session")
	}
	user := model.User{MODEL: coreAPI.MODEL{ID: accessTokenID}, Username: name, NickName: name, Status: 1}
	return sysResp.LoginResp{
		User: user, Token: accessToken, LoginType: middleware.LoginTypeAPIToken, ExpiresAt: expiresAt.UnixMilli(),
	}, nil
}

func (s *WebService) RefreshToken(c *fiber.Ctx, params *sysReq.RefreshTokenParams) (*sysResp.LoginResp, error) {
	refreshToken, err := middleware.NewJWT().ParseRefreshToken(params.RefreshToken)
	if err != nil {
		return nil, errors.New("illegal login")
	}
	if refreshToken.LoginType == middleware.LoginTypeAPIToken {
		return nil, errors.New("API Token authorization cannot be refreshed")
	}
	if err = SysService.UserService.UpdateSession(refreshToken.ID, refreshToken.SID, c.IP()); err != nil {
		return nil, errors.New("illegal login")
	}
	user, err := SysService.UserService.GetById(refreshToken.ID)
	if err != nil {
		return nil, errors.New("illegal login")
	}
	result, err := s.tokenNext(user, refreshToken.SID)
	return &result, err
}

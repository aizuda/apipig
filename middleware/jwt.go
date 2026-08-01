package middleware

import (
	"apipig/core/api/response"
	"apipig/global"
	"apipig/toolkit/snowflake"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

const (
	LoginTypeAccount  = "account"
	LoginTypeAPIToken = "api_token"
)

type TokenClaims struct {
	SID           string       `json:"sid"`
	ID            snowflake.ID `json:"id"`
	Username      string       `json:"username"`
	NickName      string       `json:"nickName"`
	LoginType     string       `json:"loginType"`
	AccessTokenID snowflake.ID `json:"accessTokenId"`
	jwt.RegisteredClaims
}

type RefreshTokenClaims struct {
	SID           string       `json:"sid"`
	ID            snowflake.ID `json:"id"`
	LoginType     string       `json:"loginType"`
	AccessTokenID snowflake.ID `json:"accessTokenId"`
	jwt.RegisteredClaims
}

func JWTAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var token string
		headerValues := c.GetReqHeaders()["accessToken"]
		if len(headerValues) == 0 {
			headerValues = c.GetReqHeaders()["Accesstoken"]
		}
		if len(headerValues) > 0 {
			token = headerValues[0]
		} else {
			token = c.Query("accessToken")
		}
		if token == "" {
			return response.FailedExpiredToken(c)
		}

		claims, err := NewJWT().ParseToken(token)
		if err != nil {
			return response.FailedExpiredToken(c)
		}
		if claims.LoginType == LoginTypeAPIToken && !isAPITokenSessionAllowed(c) {
			return response.Failed(c, "API Token login can only view data for that token")
		}

		c.Locals("tokenClaims", claims)
		return c.Next()
	}
}

func isAPITokenSessionAllowed(c *fiber.Ctx) bool {
	if c.Method() != fiber.MethodPost {
		return false
	}
	return strings.HasSuffix(c.Path(), "/ai/gateway/log/page") ||
		strings.HasSuffix(c.Path(), "/ai/gateway/token/statistics")
}

type JWT struct {
	SigningKey  []byte
	ExpiresTime int64
}

func NewJWT() *JWT {
	cj := global.CONFIG.JWT
	return &JWT{[]byte(cj.SigningKey), cj.ExpiresTime}
}

func (j *JWT) CreateToken(claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.SigningKey)
}

func (j *JWT) ParseToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return j.SigningKey, nil
	})
	if err == nil {
		if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
			return claims, nil
		}
	}
	return nil, errors.New("authorization parsing failed")
}

func (j *JWT) ParseRefreshToken(refreshToken string) (*RefreshTokenClaims, error) {
	token, err := jwt.ParseWithClaims(refreshToken, &RefreshTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return j.SigningKey, nil
	})
	if err == nil {
		if claims, ok := token.Claims.(*RefreshTokenClaims); ok && token.Valid {
			return claims, nil
		}
	}
	return nil, err
}

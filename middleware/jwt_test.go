package middleware

import (
	"fmt"
	"testing"
)

func TestJwt(t *testing.T) {
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJJRCI6IjE4MTI3NjExMDkyMjkyMDc1NTIiLCJVc2VybmFtZSI6ImFkbWluIiwiTmlja05hbWUiOiIiLCJleHAiOjE3MjI5MzcwMTMsImlhdCI6MTcyMjkzNTIxM30.mfUQ3B06ADIz-ozZ85y7_jxsnfLrutG_xkI5AUbgyOU"
	jwt := &JWT{[]byte("jC4ruWCOt29eyJhbGciOiJIUzI"), 30}
	tc, err := jwt.ParseToken(token)
	if err == nil {
		fmt.Println(tc.Username)
	}
}

package jwtHelper

import (
	"github.com/golang-jwt/jwt/v4"
	"strings"
)

// IsValid 验证是否有效token
func IsValid(secret string, tokenStr string) (bool, *jwt.MapClaims) {
	prefix := "Bearer "
	tokenStr = strings.TrimPrefix(tokenStr, prefix)
	//validate token format
	if tokenStr == "" {
		return false, nil
	}
	claims := &jwt.MapClaims{}
	token, _ := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (i interface{}, err error) {
		return []byte(secret), nil
	})

	if !token.Valid {
		return false, nil
	} else {
		return true, claims
	}
}

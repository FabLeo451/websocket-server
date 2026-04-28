package auth

import (
	"ekhoes-server/config"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type CustomClaims struct {
	SessionId  string `json:"sessionId"`
	UserId     string `json:"userId"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	Roles      string `json:"roles"`
	Privileges string `json:"privileges"`
	jwt.RegisteredClaims
}

func GenerateJWT(sessionId, userId, email, name string, roles string, privileges string, expiresAt time.Time) (string, error) {

	claims := CustomClaims{
		SessionId:  sessionId,
		UserId:     userId,
		Email:      email,
		Name:       name,
		Roles:      roles,
		Privileges: privileges,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(time.Now()),
			Issuer:   "websocket-server",
			Subject:  userId,
		},
	}

	if !expiresAt.IsZero() {
		claims.ExpiresAt = jwt.NewNumericDate(expiresAt)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(config.JWTSecret()))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func DecodeJWT(tokenString string) (jwt.MapClaims, bool, error) {

	valid := true

	var jwtSecret = []byte(config.JWTSecret())

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("invalid sign algoritm: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		if !errors.Is(err, jwt.ErrTokenExpired) {
			return nil, false, fmt.Errorf("invalid token: %w", err)
		}
	}

	valid = token.Valid

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, valid, fmt.Errorf("invalid claims")
	}

	return claims, valid, nil
}

func VerifyJWT(tokenString string) (string, error) {

	claims, valid, err := DecodeJWT(tokenString)

	if err != nil {
		return "", err
	}

	sessionId, ok := claims["sessionId"].(string)
	if !ok {
		return "", fmt.Errorf("claim 'sessionId' not found or not a string")
	}

	// Check expiration
	if !valid {
		if expRaw, ok := claims["exp"].(float64); ok {
			if time.Now().Unix() > int64(expRaw) {
				return "", fmt.Errorf("token expired")
			}
		}
	}

	return sessionId, nil
}

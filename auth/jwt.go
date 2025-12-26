package auth

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type CustomClaims struct {
	SessionId string `json:"sessionId"`
	UserId    string `json:"userId"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	jwt.RegisteredClaims
}

func generateJWT(sessionId, userId, email, name string, ttl *time.Time) (string, error) {

	claims := CustomClaims{
		SessionId: sessionId,
		UserId:    userId,
		Email:     email,
		Name:      name,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(time.Now()),
			Issuer:   "websocket-server",
			Subject:  userId,
		},
	}

	if ttl != nil {
		claims.ExpiresAt = jwt.NewNumericDate(*ttl)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func DecodeJWT(tokenString string) (jwt.MapClaims, error) {

	var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("invalid sign algoritm: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}

	return claims, nil
}

func VerifyJWT(tokenString string) (string, error) {

	claims, err := DecodeJWT(tokenString)

	if err != nil {
		return "", err
	}

	sessionId, ok := claims["sessionId"].(string)
	if !ok {
		return "", fmt.Errorf("claim 'sessionId' not found or not a string")
	}

	// Check expiration
	if expRaw, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(expRaw) {
			return "", fmt.Errorf("token expired")
		}
	}

	return sessionId, nil
}

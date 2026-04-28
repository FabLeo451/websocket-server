package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type Credentials struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	Agent      string `json:"agent"`
	Platform   string `json:"platform"`
	Model      string `json:"model"`
	DeviceName string `json:"deviceName"`
	DeviceType string `json:"deviceType"`
}

func CheckAuthorization(r *http.Request) (jwt.MapClaims, error) {

	token := r.Header.Get("Authorization")

	//fmt.Printf("[createHotSpot] Authorization: %s\n", token)

	if token == "" {
		return nil, errors.New("missing Authorization header")
	}

	claims, valid, err := DecodeJWT(token)

	if err != nil || !valid {
		return nil, errors.New("invalid token")
	}

	if claims["userId"].(string) == "" {
		return nil, errors.New("missing user id in token")
	}

	return claims, nil
}

func contains(csv string, target string) bool {
	items := strings.Split(csv, ",")
	for _, item := range items {
		if strings.TrimSpace(item) == target {
			return true
		}
	}
	return false
}

func HasPrivilege(privileges string, target string) bool {
	return contains(privileges, target) || contains(privileges, "ek_admin")
}

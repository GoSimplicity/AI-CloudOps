package jwt_test

import (
	"fmt"
	"testing"

	ijwt "github.com/GoSimplicity/CloudOps/pkg/utils/jwt"
	"github.com/golang-jwt/jwt/v5"
)

func TestXxx(t *testing.T) {
	var uc ijwt.UserClaims
	key := []byte("ebe3vxIP7sblVvUHXb7ZaiMPuz4oXo0l")
	tokenStr := "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJLNW1CUEJZTlFlTldFQnZDVEU1bXNvZzNLU0dUZGhteCIsImV4cCI6MTcyNTc4MTI2NCwiVWlkIjoxOSwiU3NpZCI6ImFlN2Q1ZDlmLTczNmMtNDliMi04NjJhLTViZjM2OWUwYWQ3MCIsIlVzZXJBZ2VudCI6Ik1vemlsbGEvNS4wIChNYWNpbnRvc2g7IEludGVsIE1hYyBPUyBYIDEwXzE1XzcpIEFwcGxlV2ViS2l0LzUzNy4zNiAoS0hUTUwsIGxpa2UgR2Vja28pIENocm9tZS8xMjguMC4wLjAgU2FmYXJpLzUzNy4zNiIsIkNvbnRlbnRUeXBlIjoiYXBwbGljYXRpb24vanNvbiJ9.GpdGnR7RrrDsGeth4jsepmHuTU-Wmj7sNmNcgeliDhOIInREEcqJgwlcdPNpiNimeZmG14C1RCkHO0ELsuohCw"
	token, err := jwt.ParseWithClaims(tokenStr, &uc, func(token *jwt.Token) (interface{}, error) {
		return key, nil
	})

	if err != nil {
		t.Error(err)
	}
	if token == nil || !token.Valid {
		t.Error("token invalid")
	}
	fmt.Println(uc)
}

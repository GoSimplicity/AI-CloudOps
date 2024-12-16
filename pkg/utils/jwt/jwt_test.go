/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package jwt_test

import (
	"fmt"
	"testing"

	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils/jwt"
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

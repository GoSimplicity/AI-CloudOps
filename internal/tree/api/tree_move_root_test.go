package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	"github.com/gin-gonic/gin"
)

type moveRootService struct {
	service.TreeNodeService
	calls, nodeID, parentID int
	failure                 error
}

func (s *moveRootService) MoveNode(_ context.Context, nodeID, parentID int) error {
	s.calls++
	s.nodeID, s.parentID = nodeID, parentID
	return s.failure
}

func TestMoveTreeNodeParentPresenceAndPathIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	maxParent := int(^uint(0) >> 1)
	overflow := strconv.FormatUint(uint64(maxParent)+1, 10)
	for _, tc := range []struct {
		name, path, body string
		allowed          bool
		parent           int
	}{
		{"explicit virtual root", "7", `{"new_parent_id":0}`, true, 0},
		{"positive parent", "7", `{"new_parent_id":14}`, true, 14},
		{"largest integer parent", "7", `{"new_parent_id":` + strconv.Itoa(maxParent) + `}`, true, maxParent},
		{"body cannot redirect entity", "7?id=999&new_parent_id=999", `{"id":999,"new_parent_id":0}`, true, 0},
		{"zero body entity cannot replace path", "7", `{"id":0,"new_parent_id":0}`, true, 0},
		{"missing parent", "7", `{}`, false, 0},
		{"null parent", "7", `{"new_parent_id":null}`, false, 0},
		{"negative parent", "7", `{"new_parent_id":-1}`, false, 0},
		{"fractional parent", "7", `{"new_parent_id":0.5}`, false, 0},
		{"string parent", "7", `{"new_parent_id":"0"}`, false, 0},
		{"empty string parent", "7", `{"new_parent_id":""}`, false, 0},
		{"boolean parent", "7", `{"new_parent_id":false}`, false, 0},
		{"overflow parent", "7", `{"new_parent_id":` + overflow + `}`, false, 0},
		{"zero entity path", "0", `{"id":7,"new_parent_id":0}`, false, 0},
		{"negative entity path", "-7", `{"id":7,"new_parent_id":0}`, false, 0},
		{"malformed entity path", "bad", `{"id":7,"new_parent_id":0}`, false, 0},
		{"overflow entity path", overflow, `{"id":7,"new_parent_id":0}`, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stub := &moveRootService{}
			app := gin.New()
			NewTreeNodeHandler(stub).RegisterRouters(app)
			request := httptest.NewRequest(http.MethodPut, "/api/tree/node/move/"+tc.path, strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			app.ServeHTTP(response, request)
			var envelope struct {
				Code int `json:"code"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatal("move handler must return its normal JSON envelope")
			}
			if tc.allowed {
				if response.Code != http.StatusOK || envelope.Code != 0 || stub.calls != 1 || stub.nodeID != 7 || stub.parentID != tc.parent {
					t.Errorf("valid parent must reach the service once with trusted URI entity: http=%d code=%d calls=%d node=%d parent=%d; want parent=%d", response.Code, envelope.Code, stub.calls, stub.nodeID, stub.parentID, tc.parent)
				}
			} else if response.Code != http.StatusBadRequest || envelope.Code == 0 || stub.calls != 0 {
				t.Errorf("invalid input must not reach business logic: http=%d code=%d calls=%d", response.Code, envelope.Code, stub.calls)
			}
		})
	}
}

func TestMoveTreeNodeFormParentPresence(t *testing.T) {
	gin.SetMode(gin.TestMode)
	overflow := strconv.FormatUint(uint64(^uint(0)>>1)+1, 10)
	for _, format := range []string{"urlencoded", "multipart", "query"} {
		for _, tc := range []struct {
			name, value string
			present     bool
			allowed     bool
			parent      int
		}{
			{"explicit virtual root", "0", true, true, 0},
			{"positive parent", "14", true, true, 14},
			{"missing parent", "", false, false, 0},
			{"empty parent", "", true, false, 0},
			{"whitespace parent", " \t\n", true, false, 0},
			{"negative parent", "-1", true, false, 0},
			{"fractional parent", "0.5", true, false, 0},
			{"invalid number", "not-an-integer", true, false, 0},
			{"overflow parent", overflow, true, false, 0},
		} {
			t.Run(format+"/"+tc.name, func(t *testing.T) {
				stub := &moveRootService{}
				app := gin.New()
				NewTreeNodeHandler(stub).RegisterRouters(app)
				values := url.Values{"id": {"999"}}
				if tc.present {
					values.Set("new_parent_id", tc.value)
				}
				path := "/api/tree/node/move/7"
				request := httptest.NewRequest(http.MethodPut, path, nil)
				switch format {
				case "urlencoded":
					request = httptest.NewRequest(http.MethodPut, path, strings.NewReader(values.Encode()))
					request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				case "query":
					request = httptest.NewRequest(http.MethodPut, path+"?"+values.Encode(), nil)
				case "multipart":
					var body bytes.Buffer
					writer := multipart.NewWriter(&body)
					for name, entries := range values {
						for _, value := range entries {
							if err := writer.WriteField(name, value); err != nil {
								t.Fatal(err)
							}
						}
					}
					if err := writer.Close(); err != nil {
						t.Fatal(err)
					}
					request = httptest.NewRequest(http.MethodPut, path, &body)
					request.Header.Set("Content-Type", writer.FormDataContentType())
				}
				response := httptest.NewRecorder()
				app.ServeHTTP(response, request)
				var envelope struct {
					Code int `json:"code"`
				}
				if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
					t.Fatal("form move must preserve the normal JSON response envelope")
				}
				if tc.allowed {
					if response.Code != http.StatusOK || envelope.Code != 0 || stub.calls != 1 || stub.nodeID != 7 || stub.parentID != tc.parent {
						t.Errorf("form move must preserve explicit parent and trusted entity: http=%d code=%d calls=%d node=%d parent=%d", response.Code, envelope.Code, stub.calls, stub.nodeID, stub.parentID)
					}
				} else if response.Code != http.StatusBadRequest || envelope.Code == 0 || stub.calls != 0 {
					t.Errorf("invalid form must be rejected before service: http=%d code=%d calls=%d", response.Code, envelope.Code, stub.calls)
				}
			})
		}
	}
}

func TestMoveTreeNodeKeepsDomainRejection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &moveRootService{failure: errors.New("existing move policy rejected the target")}
	app := gin.New()
	NewTreeNodeHandler(stub).RegisterRouters(app)
	request := httptest.NewRequest(http.MethodPut, "/api/tree/node/move/7", strings.NewReader(`{"new_parent_id":14}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	app.ServeHTTP(response, request)
	var envelope struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil || envelope.Code == 0 || stub.calls != 1 || stub.nodeID != 7 || stub.parentID != 14 {
		t.Fatal("validated shape must still honor the existing service rejection")
	}
}

func TestMoveTreeNodeRejectsMalformedFormBeforeCacheReuse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct{ name, contentType, body string }{
		{"bad URL encoding", "application/x-www-form-urlencoded", "new_parent_id=0&ignored=%ZZ"},
		{"truncated multipart", "multipart/form-data; boundary=move-root", "--move-root\r\nContent-Disposition: form-data; name=\"new_parent_id\"\r\n\r\n0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stub := &moveRootService{}
			app := gin.New()
			NewTreeNodeHandler(stub).RegisterRouters(app)
			request := httptest.NewRequest(http.MethodPut, "/api/tree/node/move/7", strings.NewReader(tc.body))
			request.Header.Set("Content-Type", tc.contentType)
			response := httptest.NewRecorder()
			app.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest || stub.calls != 0 {
				t.Fatalf("a parsed parent must not hide malformed form input: http=%d calls=%d", response.Code, stub.calls)
			}
		})
	}
}

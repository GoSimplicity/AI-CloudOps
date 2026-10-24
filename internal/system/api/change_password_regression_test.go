package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
)

// changePasswordRecorder records which identity reaches the service layer (SEC-28 regression).
type changePasswordRecorder struct {
	service.UserService
	pwd     *model.ChangePasswordReq
	woUID   int
	profile *model.UpdateProfileReq
}

func (s *changePasswordRecorder) ChangePassword(_ context.Context, r *model.ChangePasswordReq) error {
	cp := *r
	s.pwd = &cp
	return nil
}
func (s *changePasswordRecorder) WriteOff(_ context.Context, uid int, _ string) error {
	s.woUID = uid
	return nil
}
func (s *changePasswordRecorder) UpdateProfile(_ context.Context, r *model.UpdateProfileReq, _ bool) error {
	cp := *r
	s.profile = &cp
	return nil
}
func (s *changePasswordRecorder) GetProfile(_ context.Context, uid int) (*model.User, error) {
	return &model.User{Model: model.Model{ID: uid}}, nil
}

func newChangePasswordTestEngine(rec *changePasswordRecorder, uid int, name string) *gin.Engine {
	h := NewUserHandler(rec, nil)
	e := gin.New()
	e.Use(func(c *gin.Context) { c.Set("user", ijwt.UserClaims{Uid: uid, Username: name}); c.Next() })
	h.RegisterRoutes(e)
	return e
}

func TestChangePasswordIgnoresClientUserID(t *testing.T) {
	const self, victim = 7, 99
	cases := []struct{ name, ct, path, body string }{
		{"json", "application/json", "/api/user/change_password",
			`{"user_id":99,"username":"victim","password":"old-pass-x","new_password":"new-pass-x","confirm_password":"new-pass-x"}`},
		{"form", "application/x-www-form-urlencoded", "/api/user/change_password",
			url.Values{"user_id": {"99"}, "username": {"victim"}, "password": {"old-pass-x"}, "new_password": {"new-pass-x"}, "confirm_password": {"new-pass-x"}}.Encode()},
		{"query+json", "application/json", "/api/user/change_password?user_id=99",
			`{"username":"victim","password":"old-pass-x","new_password":"new-pass-x","confirm_password":"new-pass-x"}`},
		{"query+form", "application/x-www-form-urlencoded", "/api/user/change_password?user_id=99",
			url.Values{"password": {"old-pass-x"}, "new_password": {"new-pass-x"}, "confirm_password": {"new-pass-x"}}.Encode()},
		{"json user_id=0", "application/json", "/api/user/change_password",
			`{"user_id":0,"username":"victim","password":"old-pass-x","new_password":"new-pass-x","confirm_password":"new-pass-x"}`},
		{"user_id and username omitted", "application/json", "/api/user/change_password",
			`{"password":"old-pass-x","new_password":"new-pass-x","confirm_password":"new-pass-x"}`},
		{"username omitted", "application/json", "/api/user/change_password",
			`{"user_id":7,"password":"old-pass-x","new_password":"new-pass-x","confirm_password":"new-pass-x"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := &changePasswordRecorder{}
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			r.Header.Set("Content-Type", tc.ct)
			newChangePasswordTestEngine(rec, self, "alice").ServeHTTP(w, r)
			if rec.pwd == nil {
				t.Fatalf("service not reached: %d %s", w.Code, w.Body.String())
			}
			t.Logf("service received UserID=%d (token uid=%d, body/query user_id=%d)", rec.pwd.UserID, self, victim)
			if w.Code != http.StatusOK {
				t.Fatalf("unexpected status %d: %s", w.Code, w.Body.String())
			}
			if rec.pwd.UserID != self {
				t.Fatalf("client user_id overrode token uid -> %d", rec.pwd.UserID)
			}
		})
	}
}

func TestWriteOffUsesTokenUID(t *testing.T) {
	rec := &changePasswordRecorder{}
	r := httptest.NewRequest(http.MethodPost, "/api/user/write_off", strings.NewReader(`{"password":"pw-x","user_id":99,"id":99}`))
	r.Header.Set("Content-Type", "application/json")
	newChangePasswordTestEngine(rec, 7, "alice").ServeHTTP(httptest.NewRecorder(), r)
	t.Logf("write_off uid=%d", rec.woUID)
	if rec.woUID != 7 {
		t.Fatalf("write_off target=%d", rec.woUID)
	}
}

func TestUpdateProfileOwnership(t *testing.T) {
	body := `{"id":99,"real_name":"x","mobile":"1","account_type":1,"home_path":"/","avatar":"data:x"}`
	// path id = other user, non-admin -> 403
	rec := &changePasswordRecorder{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/api/user/profile/update/99", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	newChangePasswordTestEngine(rec, 7, "alice").ServeHTTP(w, r)
	t.Logf("other path id: status=%d reached=%v", w.Code, rec.profile != nil)
	if w.Code != http.StatusForbidden || rec.profile != nil {
		t.Fatalf("non-admin could update other profile")
	}
	// path id = self, body id = other -> service gets self
	rec = &changePasswordRecorder{}
	r = httptest.NewRequest(http.MethodPut, "/api/user/profile/update/7", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	newChangePasswordTestEngine(rec, 7, "alice").ServeHTTP(httptest.NewRecorder(), r)
	if rec.profile == nil || rec.profile.ID != 7 {
		t.Fatalf("body id overrode path id: %+v", rec.profile)
	}
	t.Logf("self path id + body id=99: service ID=%d", rec.profile.ID)
}

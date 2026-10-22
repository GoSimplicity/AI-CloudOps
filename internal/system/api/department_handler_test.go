package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestDepartmentHTTPPersistenceAndPermissions(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "departments.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&model.Department{}, &model.User{}); err != nil {
		t.Fatal(err)
	}
	user := model.User{Username: "member", Enable: 1}
	if err = db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", jwt.UserClaims{Uid: user.ID, Username: c.GetHeader("X-Test-Identity")})
		c.Next()
	})
	NewDepartmentHandler(service.NewDepartmentService(dao.NewDepartmentDAO(db))).RegisterRouters(router)
	server := httptest.NewServer(router)
	defer server.Close()
	call := func(method, path, identity string, body any) (int, map[string]any) {
		t.Helper()
		raw, _ := json.Marshal(body)
		r, _ := http.NewRequest(method, server.URL+path, bytes.NewReader(raw))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Test-Identity", identity)
		response, err := server.Client().Do(r)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var result map[string]any
		if err = json.NewDecoder(response.Body).Decode(&result); err != nil {
			t.Fatal(err)
		}
		return response.StatusCode, result
	}
	if status, _ := call("POST", "/api/department/create", "member", map[string]any{"name": "Forbidden"}); status != 403 {
		t.Fatalf("non-admin status %d", status)
	}
	_, created := call("POST", "/api/department/create", "admin", map[string]any{"name": "Engineering"})
	if created["code"] != float64(0) {
		t.Fatal(created)
	}
	id := int(created["data"].(map[string]any)["id"].(float64))
	_, child := call("POST", "/api/department/create", "admin", map[string]any{"name": "Operations", "parent_id": id})
	if child["code"] != float64(0) {
		t.Fatal(child)
	}
	childID := int(child["data"].(map[string]any)["id"].(float64))
	_, cycle := call("PUT", fmt.Sprintf("/api/department/%d/update", id), "admin", map[string]any{"name": "Engineering", "parent_id": childID})
	if cycle["code"] == float64(0) {
		t.Fatal("cycle accepted")
	}
	_, nonempty := call("DELETE", fmt.Sprintf("/api/department/%d/delete", id), "admin", nil)
	if nonempty["code"] == float64(0) {
		t.Fatal("deleted parent with child")
	}
	if status, _ := call("PUT", fmt.Sprintf("/api/department/user/%d", user.ID), "member", map[string]int{"department_id": childID}); status != 403 {
		t.Fatal("non-admin assigned department")
	}
	_, assigned := call("PUT", fmt.Sprintf("/api/department/user/%d", user.ID), "admin", map[string]int{"department_id": childID})
	if assigned["code"] != float64(0) {
		t.Fatal(assigned)
	}
	var persisted model.User
	db.First(&persisted, user.ID)
	if persisted.DepartmentID != childID {
		t.Fatal("assignment not persisted")
	}
	_, nonempty = call("DELETE", fmt.Sprintf("/api/department/%d/delete", childID), "admin", nil)
	if nonempty["code"] == float64(0) {
		t.Fatal("deleted department with member")
	}
	_, cleared := call("PUT", fmt.Sprintf("/api/department/user/%d", user.ID), "admin", map[string]int{"department_id": 0})
	if cleared["code"] != float64(0) {
		t.Fatal(cleared)
	}
	for _, n := range []int{childID, id} {
		_, deleted := call("DELETE", fmt.Sprintf("/api/department/%d/delete", n), "admin", nil)
		if deleted["code"] != float64(0) {
			t.Fatal(deleted)
		}
	}
	_, listed := call("GET", "/api/department/list", "admin", nil)
	if listed["data"].(map[string]any)["total"] != float64(0) {
		t.Fatal("deleted departments remain listed")
	}
}

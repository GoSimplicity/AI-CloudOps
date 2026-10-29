package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/middleware"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	systemservice "github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type editActionRoles struct {
	systemservice.RoleService
	grants []*model.Api
}

func (r editActionRoles) GetUserRoles(context.Context, int) (*model.ListResp[*model.Role], error) {
	return &model.ListResp[*model.Role]{Items: []*model.Role{{Status: 1, Apis: r.grants}}}, nil
}

func TestEditActionMatchesUpdateStateAndRequiresUpdateRBAC(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "actions.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err = db.AutoMigrate(&model.WorkorderInstance{}, &model.WorkorderProcess{}, &model.WorkorderFormDesign{}, &model.WorkorderCategory{}, &model.WorkorderInstanceComment{}, &model.WorkorderInstanceFlow{}, &model.WorkorderInstanceTimeline{}); err != nil {
		t.Fatal(err)
	}
	process := model.WorkorderProcess{Name: "action fixture", Definition: model.JSONMap{"steps": []any{map[string]any{"id": "restricted", "type": "task", "assignee_type": "user", "assignee_ids": []int{7}, "actions": []string{"edit"}}}}}
	if err = db.Create(&process).Error; err != nil {
		t.Fatal(err)
	}
	step := "restricted"
	instance := model.WorkorderInstance{Title: "fixture", SerialNumber: "fixture", ProcessID: process.ID, CurrentStepID: &step, Status: model.InstanceStatusDraft, OperatorID: 7, OperatorName: "creator"}
	if err = db.Create(&instance).Error; err != nil {
		t.Fatal(err)
	}
	svc := service.NewInstanceService(dao.NewWorkorderInstanceDAO(db, zap.NewNop()), nil, nil, nil, dao.NewProcessDAO(db, zap.NewNop()), nil, nil, nil, zap.NewNop())
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name     string
		status   int8
		editable bool
	}{
		{"draft", model.InstanceStatusDraft, true}, {"pending", model.InstanceStatusPending, true},
		{"processing", model.InstanceStatusProcessing, false}, {"completed", model.InstanceStatusCompleted, false},
		{"rejected", model.InstanceStatusRejected, false}, {"cancelled", model.InstanceStatusCancelled, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := db.Model(&model.WorkorderInstance{}).Where("id = ?", instance.ID).Update("status", tc.status).Error; err != nil {
				t.Fatal(err)
			}
			for _, granted := range []bool{false, true} {
				grants := []*model.Api{{Method: 1, Path: "/api/workorder/instance/actions/:id"}}
				if granted {
					grants = append(grants, &model.Api{Method: 3, Path: "/api/workorder/instance/update/:id"})
				}
				engine := gin.New()
				// User 8 is not the creator or configured step assignee. A forged operator_id must not change that.
				engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 8, Username: "operator"}); c.Next() })
				engine.Use(middleware.NewAuthMiddleware(editActionRoles{grants: grants}).CheckAuth())
				NewInstanceHandler(svc).RegisterRouters(engine)
				rec := httptest.NewRecorder()
				engine.ServeHTTP(rec, httptest.NewRequest("GET", fmt.Sprintf("/api/workorder/instance/actions/%d?operator_id=7", instance.ID), nil))
				var result struct {
					Code int      `json:"code"`
					Data []string `json:"data"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil || rec.Code != 200 || result.Code != 0 {
					t.Fatalf("actions: %d %s", rec.Code, rec.Body.String())
				}
				if slices.Contains(result.Data, "edit") != tc.editable {
					t.Fatalf("status %d: actions %v", tc.status, result.Data)
				}
				if slices.Contains(result.Data, "complete") {
					t.Fatal("forged operator gained workflow action")
				}
				update := httptest.NewRequest("PUT", fmt.Sprintf("/api/workorder/instance/update/%d", instance.ID), strings.NewReader(`{"title":"edited","operator_id":7}`))
				update.Header.Set("Content-Type", "application/json")
				rec = httptest.NewRecorder()
				engine.ServeHTTP(rec, update)
				if !granted {
					if rec.Code != 403 {
						t.Fatalf("ungranted update: %d %s", rec.Code, rec.Body.String())
					}
					continue
				}
				var outcome struct {
					Code int `json:"code"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &outcome); err != nil {
					t.Fatal(err)
				}
				if (outcome.Code == 0) != tc.editable {
					t.Fatalf("update status %d: %s", tc.status, rec.Body.String())
				}
			}
			// The configured step itself contains "edit". It cannot bypass the state
			// restriction even for an assignee allowed to perform workflow actions.
			engine := gin.New()
			engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 7}); c.Next() })
			engine.Use(middleware.NewAuthMiddleware(editActionRoles{grants: []*model.Api{{Method: 1, Path: "/api/workorder/instance/actions/:id"}}}).CheckAuth())
			NewInstanceHandler(svc).RegisterRouters(engine)
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, httptest.NewRequest("GET", fmt.Sprintf("/api/workorder/instance/actions/%d", instance.ID), nil))
			var result struct {
				Code int      `json:"code"`
				Data []string `json:"data"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil || rec.Code != 200 || result.Code != 0 {
				t.Fatalf("assignee actions: %d %s", rec.Code, rec.Body.String())
			}
			if slices.Contains(result.Data, "edit") != tc.editable {
				t.Fatalf("assignee status %d: %v", tc.status, result.Data)
			}
			if tc.status == model.InstanceStatusProcessing && !slices.Contains(result.Data, "complete") {
				t.Fatal("assignee lost workflow action")
			}
		})
	}
}

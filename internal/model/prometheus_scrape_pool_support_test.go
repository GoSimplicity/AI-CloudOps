package model

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestScrapePoolSupportGinBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, operation := range []string{"create", "update"} {
		for _, field := range []string{"support_alert", "support_record"} {
			for _, value := range []interface{}{nil, 0, 1, 2, -1, 3, 127} {
				t.Run(fmt.Sprintf("%s/%s/%v", operation, field, value), func(t *testing.T) {
					payload := map[string]interface{}{"name": "fixture", "support_alert": 2, "support_record": 2}
					if value == nil {
						delete(payload, field)
					} else {
						payload[field] = value
					}
					raw, err := json.Marshal(payload)
					if err != nil {
						t.Fatal(err)
					}
					router := gin.New()
					bound := false
					router.POST("/bind", func(ctx *gin.Context) {
						var err error
						if operation == "create" {
							var req CreateMonitorScrapePoolReq
							err = ctx.ShouldBindJSON(&req)
						} else {
							req := UpdateMonitorScrapePoolReq{ID: 7}
							err = ctx.ShouldBindJSON(&req)
						}
						if err != nil {
							ctx.Status(400)
							return
						}
						bound = true
						ctx.Status(204)
					})
					response := httptest.NewRecorder()
					request := httptest.NewRequest("POST", "/bind", bytes.NewReader(raw))
					request.Header.Set("Content-Type", "application/json")
					router.ServeHTTP(response, request)
					want := value == 1 || value == 2 || operation == "create" && (value == nil || value == 0)
					if bound != want || want && response.Code != 204 || !want && response.Code != 400 {
						t.Fatalf("binding acceptance=%v status=%d, want acceptance=%v", bound, response.Code, want)
					}
				})
			}
		}
	}
}

type poolDefaultsNoIO struct{ calls int }

func (p *poolDefaultsNoIO) PrepareContext(context.Context, string) (*sql.Stmt, error) {
	p.calls++
	panic("unexpected database I/O in dry-run default test")
}
func (p *poolDefaultsNoIO) ExecContext(context.Context, string, ...interface{}) (sql.Result, error) {
	p.calls++
	panic("unexpected database I/O in dry-run default test")
}
func (p *poolDefaultsNoIO) QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error) {
	p.calls++
	panic("unexpected database I/O in dry-run default test")
}
func (p *poolDefaultsNoIO) QueryRowContext(context.Context, string, ...interface{}) *sql.Row {
	p.calls++
	panic("unexpected database I/O in dry-run default test")
}
func TestScrapePoolActualGORMCreateDefaultsRemainDisabled(t *testing.T) {
	connection := &poolDefaultsNoIO{}
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: connection, SkipInitializeWithVersion: true}), &gorm.Config{DryRun: true, SkipDefaultTransaction: true, DisableAutomaticPing: true, Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	for _, alert := range []int8{0, 1, 2} {
		for _, record := range []int8{0, 1, 2} {
			t.Run(fmt.Sprintf("%d/%d", alert, record), func(t *testing.T) {
				row := &MonitorScrapePool{Name: "fixture", SupportAlert: alert, SupportRecord: record}
				result := db.Create(row)
				if result.Error != nil {
					t.Fatal(result.Error)
				}
				expectedAlert, expectedRecord := alert, record
				if alert == 0 {
					expectedAlert = 2
				}
				if record == 0 {
					expectedRecord = 2
				}
				if row.SupportAlert != expectedAlert || row.SupportRecord != expectedRecord || result.Statement.SQL.Len() == 0 || len(result.Statement.Vars) == 0 {
					t.Fatal("actual GORM create callbacks changed omitted/default support semantics")
				}
			})
		}
	}
	if connection.calls != 0 {
		t.Fatal("default verification contacted a database")
	}
}

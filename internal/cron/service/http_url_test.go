package service

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"testing"
)

func TestHTTPJobURLContractOnCreateAndUpdate(t *testing.T) {
	for _, tc := range []struct {
		name, url string
		valid     bool
	}{
		{"single digit loopback port", "http://127.0.0.1:9/fixture", true}, {"https", "https://example.invalid/path", true}, {"uppercase scheme", "HTTP://localhost:9", true}, {"ipv6", "http://[::1]:9", true},
		{"empty", "", false}, {"ftp", "ftp://example.invalid/path", false}, {"empty host", "http:///path", false}, {"zero port", "http://localhost:0", false}, {"overflow port", "http://localhost:65536", false}, {"negative port", "http://localhost:-1", false}, {"non numeric port", "http://localhost:abc", false}, {"newline", "http://local\nhost:9", false}, {"tab", "http://local\thost:9", false}, {"leading space", " http://localhost:9", false}, {"space in path", "http://localhost/a b", false}, {"backslash", "http:\\localhost:9", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &localExecStore{job: &model.CronJob{Model: model.Model{ID: 1}, JobType: model.CronJobTypeHTTP, Status: model.CronJobStatusDisabled, Schedule: "@every 1h"}}
			service := newLocalExecService(false, store, nil)
			create := createReq(model.CronJobTypeHTTP)
			create.HTTPUrl = tc.url
			if err := service.CreateCronJob(context.Background(), create); (err == nil) != tc.valid || (store.created != nil) != tc.valid {
				t.Fatalf("create validity=%v err=%v", tc.valid, err)
			}
			update := updateReq(model.CronJobTypeHTTP)
			update.HTTPUrl = tc.url
			if err := service.UpdateCronJob(context.Background(), update); (err == nil) != tc.valid || (store.updated != nil) != tc.valid {
				t.Fatalf("update validity=%v err=%v", tc.valid, err)
			}
		})
	}
}

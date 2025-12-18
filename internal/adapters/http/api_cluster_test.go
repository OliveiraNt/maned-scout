package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OliveiraNt/kdash/internal/config"
)

func TestAPIClusters_CRUD(t *testing.T) {
	s := buildServer(t)

	// Add cluster
	body, _ := json.Marshal(config.ClusterConfig{Name: "dev", Brokers: []string{"localhost:9092"}})
	req := httptest.NewRequest(http.MethodPost, "/api/clusters", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	s.apiAddCluster(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	// List clusters
	rec = httptest.NewRecorder()
	s.apiListClusters(rec, httptest.NewRequest(http.MethodGet, "/api/clusters", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var list []config.ClusterConfig
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if len(list) != 1 || list[0].Name != "dev" {
		t.Fatalf("unexpected list: %+v", list)
	}

	// Update cluster
	upd, _ := json.Marshal(config.ClusterConfig{Brokers: []string{"localhost:9093"}})
	req = httptest.NewRequest(http.MethodPut, "/api/clusters/dev", bytes.NewReader(upd))
	rec = httptest.NewRecorder()
	ctx := chiCtxWithParam("clusterName", "dev", req)
	s.apiUpdateCluster(rec, req.WithContext(ctx))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	cfg, ok := s.clusterService.GetCluster("dev")
	if !ok || len(cfg.Brokers) != 1 || cfg.Brokers[0] != "localhost:9093" {
		t.Fatalf("update not applied: %+v", cfg)
	}

	// Delete cluster
	req = httptest.NewRequest(http.MethodDelete, "/api/clusters/dev", nil)
	rec = httptest.NewRecorder()
	ctx = chiCtxWithParam("clusterName", "dev", req)
	s.apiDeleteCluster(rec, req.WithContext(ctx))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if _, ok := s.clusterService.GetCluster("dev"); ok {
		t.Fatalf("cluster should be deleted")
	}
}

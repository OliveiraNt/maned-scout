package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OliveiraNt/maned-scout/internal/config"
)

func TestUIHome(t *testing.T) {
	s := buildServer(t)
	// Test with no clusters
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	s.uiHome(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("expected content-type text/html, got %s", rec.Header().Get("Content-Type"))
	}

	// Add a cluster and test
	body, _ := json.Marshal(config.ClusterConfig{Name: "dev", Brokers: []string{"localhost:9092"}})
	s.apiAddCluster(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/clusters", bytes.NewReader(body)))

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	s.uiHome(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestUIClusterDetail(t *testing.T) {
	s := buildServer(t)

	// Test with non-existent cluster
	req := httptest.NewRequest(http.MethodGet, "/clusters/nonexistent", nil)
	rec := httptest.NewRecorder()
	ctx := chiCtxWithParam("clusterName", "nonexistent", req)
	s.uiClusterDetail(rec, req.WithContext(ctx))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}

	// Add a cluster and test
	body, _ := json.Marshal(config.ClusterConfig{Name: "dev", Brokers: []string{"localhost:9092"}})
	s.apiAddCluster(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/clusters", bytes.NewReader(body)))

	req = httptest.NewRequest(http.MethodGet, "/clusters/dev", nil)
	rec = httptest.NewRecorder()
	ctx = chiCtxWithParam("clusterName", "dev", req)
	s.uiClusterDetail(rec, req.WithContext(ctx))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("expected content-type text/html, got %s", rec.Header().Get("Content-Type"))
	}
}

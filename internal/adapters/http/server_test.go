package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OliveiraNt/kdash/internal/application"
	"github.com/OliveiraNt/kdash/internal/config"
	"github.com/OliveiraNt/kdash/internal/domain"
	"github.com/OliveiraNt/kdash/internal/infrastructure/repository"
	"github.com/OliveiraNt/kdash/internal/registry"
	"github.com/go-chi/chi/v5"
)

type testClient struct {
	healthy bool
	topics  map[string]int
}

func (c *testClient) IsHealthy() bool                                      { return c.healthy }
func (c *testClient) ListTopics(showInternal bool) (map[string]int, error) { return c.topics, nil }
func (c *testClient) GetClusterInfo() (*domain.Cluster, error)             { return nil, nil }
func (c *testClient) GetClusterStats() (*domain.ClusterStats, error) {
	return &domain.ClusterStats{TotalTopics: len(c.topics)}, nil
}
func (c *testClient) GetBrokerDetails() ([]domain.BrokerDetail, error)           { return nil, nil }
func (c *testClient) ListConsumerGroups() ([]domain.ConsumerGroupSummary, error) { return nil, nil }
func (c *testClient) Close()                                                     {}

type testFactory struct{}

func (f testFactory) CreateClient(cfg config.ClusterConfig) (domain.KafkaClient, error) {
	// Each cluster gets a client marked healthy with no topics by default
	return &testClient{healthy: true, topics: map[string]int{}}, nil
}

// helper to build server and chi router without starting network listener
func buildServer(t *testing.T) *Server {
	t.Helper()
	tdir := t.TempDir()
	cfgPath := tdir + "/config.yml"
	repo := repository.NewClusterRepository(cfgPath, testFactory{})
	svc := application.NewClusterService(repo, testFactory{})
	registry.InitLogger()
	return New(svc, repo)
}

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
	ctx := chiCtxWithParam("name", "dev", req)
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
	ctx = chiCtxWithParam("name", "dev", req)
	s.apiDeleteCluster(rec, req.WithContext(ctx))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if _, ok := s.clusterService.GetCluster("dev"); ok {
		t.Fatalf("cluster should be deleted")
	}
}

func TestAPIListTopics(t *testing.T) {
	s := buildServer(t)
	// Add a cluster via API to ensure client exists in repo
	body, _ := json.Marshal(config.ClusterConfig{Name: "dev", Brokers: []string{"localhost:9092"}})
	s.apiAddCluster(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/clusters", bytes.NewReader(body)))

	// Inject topics into existing client
	if cl, ok := s.repo.GetClient("dev"); ok {
		if tc, ok2 := cl.(*testClient); ok2 {
			tc.topics = map[string]int{"a": 1, "b": 3}
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/cluster/dev/topics", nil)
	rec := httptest.NewRecorder()
	ctx := chiCtxWithParam("name", "dev", req)
	s.apiListTopics(rec, req.WithContext(ctx))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var topics map[string]int
	if err := json.Unmarshal(rec.Body.Bytes(), &topics); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if len(topics) != 2 || topics["a"] != 1 || topics["b"] != 3 {
		t.Fatalf("unexpected topics: %+v", topics)
	}
}

// chiCtxWithParam adds a single URL param to request context for handler funcs using chi.URLParam
func chiCtxWithParam(key, val string, req *http.Request) context.Context {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
}

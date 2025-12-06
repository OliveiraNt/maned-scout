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

func (c *testClient) IsHealthy() bool                           { return c.healthy }
func (c *testClient) ListTopics(_ bool) (map[string]int, error) { return c.topics, nil }
func (c *testClient) GetClusterInfo() (*domain.Cluster, error)  { return nil, nil }
func (c *testClient) GetClusterStats() (*domain.ClusterStats, error) {
	return &domain.ClusterStats{TotalTopics: len(c.topics)}, nil
}
func (c *testClient) GetBrokerDetails() ([]domain.BrokerDetail, error)           { return nil, nil }
func (c *testClient) ListConsumerGroups() ([]domain.ConsumerGroupSummary, error) { return nil, nil }
func (c *testClient) GetTopicDetail(_ string) (*domain.TopicDetail, error) {
	return &domain.TopicDetail{
		Name:              "test-topic",
		Partitions:        3,
		ReplicationFactor: 1,
		Configs:           map[string]string{},
		PartitionDetails:  []domain.PartitionDetail{},
	}, nil
}
func (c *testClient) CreateTopic(_ domain.CreateTopicRequest) error { return nil }
func (c *testClient) DeleteTopic(_ string) error                    { return nil }
func (c *testClient) UpdateTopicConfig(_ string, _ domain.UpdateTopicConfigRequest) error {
	return nil
}
func (c *testClient) IncreasePartitions(_ string, _ domain.IncreasePartitionsRequest) error {
	return nil
}
func (c *testClient) Close() {}

type testFactory struct{}

func (f testFactory) CreateClient(_ config.ClusterConfig) (domain.KafkaClient, error) {
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

// Helper to add multiple URL params
func chiCtxWithParams(params map[string]string, req *http.Request) context.Context {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
}

// ==================== Topic API Tests ====================

func TestAPIGetTopicDetail(t *testing.T) {
	s := buildServer(t)
	// Add a cluster via API
	body, _ := json.Marshal(config.ClusterConfig{Name: "dev", Brokers: []string{"localhost:9092"}})
	s.apiAddCluster(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/clusters", bytes.NewReader(body)))

	// Test successful topic detail retrieval
	req := httptest.NewRequest(http.MethodGet, "/api/cluster/dev/topics/test-topic", nil)
	rec := httptest.NewRecorder()
	ctx := chiCtxWithParams(map[string]string{"name": "dev", "topic": "test-topic"}, req)
	s.apiGetTopicDetail(rec, req.WithContext(ctx))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Test with non-existent cluster
	req = httptest.NewRequest(http.MethodGet, "/api/cluster/nonexistent/topics/test-topic", nil)
	rec = httptest.NewRecorder()
	ctx = chiCtxWithParams(map[string]string{"name": "nonexistent", "topic": "test-topic"}, req)
	s.apiGetTopicDetail(rec, req.WithContext(ctx))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestAPICreateTopic(t *testing.T) {
	s := buildServer(t)
	// Add a cluster via API
	body, _ := json.Marshal(config.ClusterConfig{Name: "dev", Brokers: []string{"localhost:9092"}})
	s.apiAddCluster(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/clusters", bytes.NewReader(body)))

	// Test successful topic creation
	createReq := domain.CreateTopicRequest{
		Name:              "new-topic",
		NumPartitions:     3,
		ReplicationFactor: 1,
	}
	body, _ = json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/api/cluster/dev/topics", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	ctx := chiCtxWithParam("name", "dev", req)
	s.apiCreateTopic(rec, req.WithContext(ctx))
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	// Test with missing topic name
	badReq := domain.CreateTopicRequest{
		Name:              "",
		NumPartitions:     3,
		ReplicationFactor: 1,
	}
	body, _ = json.Marshal(badReq)
	req = httptest.NewRequest(http.MethodPost, "/api/cluster/dev/topics", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	ctx = chiCtxWithParam("name", "dev", req)
	s.apiCreateTopic(rec, req.WithContext(ctx))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty name, got %d", rec.Code)
	}

	// Test with invalid partitions
	badReq = domain.CreateTopicRequest{
		Name:              "topic",
		NumPartitions:     0,
		ReplicationFactor: 1,
	}
	body, _ = json.Marshal(badReq)
	req = httptest.NewRequest(http.MethodPost, "/api/cluster/dev/topics", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	ctx = chiCtxWithParam("name", "dev", req)
	s.apiCreateTopic(rec, req.WithContext(ctx))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for zero partitions, got %d", rec.Code)
	}

	// Test with invalid replication factor
	badReq = domain.CreateTopicRequest{
		Name:              "topic",
		NumPartitions:     3,
		ReplicationFactor: 0,
	}
	body, _ = json.Marshal(badReq)
	req = httptest.NewRequest(http.MethodPost, "/api/cluster/dev/topics", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	ctx = chiCtxWithParam("name", "dev", req)
	s.apiCreateTopic(rec, req.WithContext(ctx))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for zero replication factor, got %d", rec.Code)
	}

	// Test with non-existent cluster
	body, _ = json.Marshal(createReq)
	req = httptest.NewRequest(http.MethodPost, "/api/cluster/nonexistent/topics", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	ctx = chiCtxWithParam("name", "nonexistent", req)
	s.apiCreateTopic(rec, req.WithContext(ctx))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestAPIDeleteTopic(t *testing.T) {
	s := buildServer(t)
	// Add a cluster via API
	body, _ := json.Marshal(config.ClusterConfig{Name: "dev", Brokers: []string{"localhost:9092"}})
	s.apiAddCluster(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/clusters", bytes.NewReader(body)))

	// Test successful topic deletion
	req := httptest.NewRequest(http.MethodDelete, "/api/cluster/dev/topics/test-topic", nil)
	rec := httptest.NewRecorder()
	ctx := chiCtxWithParams(map[string]string{"name": "dev", "topic": "test-topic"}, req)
	s.apiDeleteTopic(rec, req.WithContext(ctx))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}

	// Test with non-existent cluster
	req = httptest.NewRequest(http.MethodDelete, "/api/cluster/nonexistent/topics/test-topic", nil)
	rec = httptest.NewRecorder()
	ctx = chiCtxWithParams(map[string]string{"name": "nonexistent", "topic": "test-topic"}, req)
	s.apiDeleteTopic(rec, req.WithContext(ctx))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestAPIUpdateTopicConfig(t *testing.T) {
	s := buildServer(t)
	// Add a cluster via API
	body, _ := json.Marshal(config.ClusterConfig{Name: "dev", Brokers: []string{"localhost:9092"}})
	s.apiAddCluster(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/clusters", bytes.NewReader(body)))

	// Test successful config update
	val := "1000"
	updateReq := domain.UpdateTopicConfigRequest{
		Configs: map[string]*string{
			"retention.ms": &val,
		},
	}
	body, _ = json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPut, "/api/cluster/dev/topics/test-topic/config", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	ctx := chiCtxWithParams(map[string]string{"name": "dev", "topic": "test-topic"}, req)
	s.apiUpdateTopicConfig(rec, req.WithContext(ctx))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Test with empty configs
	emptyReq := domain.UpdateTopicConfigRequest{
		Configs: map[string]*string{},
	}
	body, _ = json.Marshal(emptyReq)
	req = httptest.NewRequest(http.MethodPut, "/api/cluster/dev/topics/test-topic/config", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	ctx = chiCtxWithParams(map[string]string{"name": "dev", "topic": "test-topic"}, req)
	s.apiUpdateTopicConfig(rec, req.WithContext(ctx))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty configs, got %d", rec.Code)
	}

	// Test with non-existent cluster
	body, _ = json.Marshal(updateReq)
	req = httptest.NewRequest(http.MethodPut, "/api/cluster/nonexistent/topics/test-topic/config", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	ctx = chiCtxWithParams(map[string]string{"name": "nonexistent", "topic": "test-topic"}, req)
	s.apiUpdateTopicConfig(rec, req.WithContext(ctx))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestAPIIncreasePartitions(t *testing.T) {
	s := buildServer(t)
	// Add a cluster via API
	body, _ := json.Marshal(config.ClusterConfig{Name: "dev", Brokers: []string{"localhost:9092"}})
	s.apiAddCluster(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/clusters", bytes.NewReader(body)))

	// Test successful partition increase
	increaseReq := domain.IncreasePartitionsRequest{
		TotalPartitions: 5,
	}
	body, _ = json.Marshal(increaseReq)
	req := httptest.NewRequest(http.MethodPost, "/api/cluster/dev/topics/test-topic/partitions", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	ctx := chiCtxWithParams(map[string]string{"name": "dev", "topic": "test-topic"}, req)
	s.apiIncreasePartitions(rec, req.WithContext(ctx))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Test with invalid partitions
	badReq := domain.IncreasePartitionsRequest{
		TotalPartitions: 0,
	}
	body, _ = json.Marshal(badReq)
	req = httptest.NewRequest(http.MethodPost, "/api/cluster/dev/topics/test-topic/partitions", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	ctx = chiCtxWithParams(map[string]string{"name": "dev", "topic": "test-topic"}, req)
	s.apiIncreasePartitions(rec, req.WithContext(ctx))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for zero partitions, got %d", rec.Code)
	}

	// Test with non-existent cluster
	body, _ = json.Marshal(increaseReq)
	req = httptest.NewRequest(http.MethodPost, "/api/cluster/nonexistent/topics/test-topic/partitions", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	ctx = chiCtxWithParams(map[string]string{"name": "nonexistent", "topic": "test-topic"}, req)
	s.apiIncreasePartitions(rec, req.WithContext(ctx))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

// ==================== UI Handler Tests ====================

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
	req := httptest.NewRequest(http.MethodGet, "/cluster/nonexistent", nil)
	rec := httptest.NewRecorder()
	ctx := chiCtxWithParam("name", "nonexistent", req)
	s.uiClusterDetail(rec, req.WithContext(ctx))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}

	// Add a cluster and test
	body, _ := json.Marshal(config.ClusterConfig{Name: "dev", Brokers: []string{"localhost:9092"}})
	s.apiAddCluster(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/clusters", bytes.NewReader(body)))

	req = httptest.NewRequest(http.MethodGet, "/cluster/dev", nil)
	rec = httptest.NewRecorder()
	ctx = chiCtxWithParam("name", "dev", req)
	s.uiClusterDetail(rec, req.WithContext(ctx))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("expected content-type text/html, got %s", rec.Header().Get("Content-Type"))
	}
}

func TestUITopicsList(t *testing.T) {
	s := buildServer(t)

	// Test with non-existent cluster
	req := httptest.NewRequest(http.MethodGet, "/cluster/nonexistent/topics", nil)
	rec := httptest.NewRecorder()
	ctx := chiCtxWithParam("name", "nonexistent", req)
	s.uiTopicsList(rec, req.WithContext(ctx))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}

	// Add a cluster and test
	body, _ := json.Marshal(config.ClusterConfig{Name: "dev", Brokers: []string{"localhost:9092"}})
	s.apiAddCluster(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/clusters", bytes.NewReader(body)))

	// Test without showInternal parameter
	req = httptest.NewRequest(http.MethodGet, "/cluster/dev/topics", nil)
	rec = httptest.NewRecorder()
	ctx = chiCtxWithParam("name", "dev", req)
	s.uiTopicsList(rec, req.WithContext(ctx))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Test with showInternal=true parameter
	req = httptest.NewRequest(http.MethodGet, "/cluster/dev/topics?showInternal=true", nil)
	rec = httptest.NewRecorder()
	ctx = chiCtxWithParam("name", "dev", req)
	s.uiTopicsList(rec, req.WithContext(ctx))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestUITopicDetail(t *testing.T) {
	s := buildServer(t)

	// Test with non-existent cluster
	req := httptest.NewRequest(http.MethodGet, "/cluster/nonexistent/topics/test-topic", nil)
	rec := httptest.NewRecorder()
	ctx := chiCtxWithParams(map[string]string{"name": "nonexistent", "topic": "test-topic"}, req)
	s.uiTopicDetail(rec, req.WithContext(ctx))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}

	// Add a cluster and test
	body, _ := json.Marshal(config.ClusterConfig{Name: "dev", Brokers: []string{"localhost:9092"}})
	s.apiAddCluster(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/clusters", bytes.NewReader(body)))

	// Test successful topic detail
	req = httptest.NewRequest(http.MethodGet, "/cluster/dev/topics/test-topic", nil)
	rec = httptest.NewRecorder()
	ctx = chiCtxWithParams(map[string]string{"name": "dev", "topic": "test-topic"}, req)
	s.uiTopicDetail(rec, req.WithContext(ctx))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("expected content-type text/html, got %s", rec.Header().Get("Content-Type"))
	}
}

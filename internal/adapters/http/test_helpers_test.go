package httpserver

import (
	"context"
	"net/http"
	"testing"

	"github.com/OliveiraNt/kdash/internal/application"
	"github.com/OliveiraNt/kdash/internal/config"
	"github.com/OliveiraNt/kdash/internal/domain"
	"github.com/OliveiraNt/kdash/internal/infrastructure/repository"
	"github.com/OliveiraNt/kdash/internal/registry"
	"github.com/go-chi/chi/v5"
)

// testClient implements domain.KafkaClient for tests
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
func (c *testClient) StreamMessages(_ context.Context, _ string, _ chan<- domain.Message) {}
func (c *testClient) WriteMessage(_ context.Context, _ string, _ domain.Message)          {}
func (c *testClient) Close()                                                              {}

type testFactory struct{}

func (f testFactory) CreateClient(_ config.ClusterConfig) (domain.KafkaClient, error) {
	// Each cluster gets a client marked healthy with no topics by default
	return &testClient{healthy: true, topics: map[string]int{}}, nil
}

// buildServer builds a Server instance for tests with a temporary config file
func buildServer(t *testing.T) *Server {
	t.Helper()
	tdir := t.TempDir()
	cfgPath := tdir + "/config.yml"
	repo := repository.NewClusterRepository(cfgPath, testFactory{})
	svc := application.NewClusterService(repo, testFactory{})
	registry.InitLogger()
	return New(svc, repo)
}

// chiCtxWithParam adds a single URL param to request context for handler funcs using chi.URLParam
func chiCtxWithParam(key, val string, req *http.Request) context.Context {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
}

// chiCtxWithParams adds multiple URL params to request context
func chiCtxWithParams(params map[string]string, req *http.Request) context.Context {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
}

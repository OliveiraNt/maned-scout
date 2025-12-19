package httpserver

import (
	"context"
	"net/http"
	"testing"

	"github.com/OliveiraNt/maned-scout/internal/application"
	"github.com/OliveiraNt/maned-scout/internal/config"
	"github.com/OliveiraNt/maned-scout/internal/domain"
	"github.com/OliveiraNt/maned-scout/internal/infrastructure/repository"
	"github.com/OliveiraNt/maned-scout/internal/utils"
	"github.com/go-chi/chi/v5"
	"github.com/twmb/franz-go/pkg/kadm"
)

// mockClient implements domain.KafkaClient for tests
type mockClient struct {
	healthy bool
	topics  map[string]int
}

func (c *mockClient) IsHealthy() bool                           { return c.healthy }
func (c *mockClient) ListTopics(_ bool) (map[string]int, error) { return c.topics, nil }
func (c *mockClient) GetClusterInfo() (*domain.Cluster, error)  { return nil, nil }
func (c *mockClient) GetClusterStats() (*domain.ClusterStats, error) {
	return &domain.ClusterStats{TotalTopics: len(c.topics)}, nil
}
func (c *mockClient) GetBrokerDetails() ([]domain.BrokerDetail, error)           { return nil, nil }
func (c *mockClient) ListConsumerGroups() ([]domain.ConsumerGroupSummary, error) { return nil, nil }
func (c *mockClient) ListConsumerGroupsWithLagFromTopic(ctx context.Context, topicName string) (kadm.DescribedGroupLags, error) {
	return nil, nil
}
func (c *mockClient) GetTopicDetail(_ string) (*domain.TopicDetail, error) {
	return &domain.TopicDetail{
		Name:              "test-topic",
		Partitions:        3,
		ReplicationFactor: 1,
		Configs:           map[string]string{},
		PartitionDetails:  []domain.PartitionDetail{},
	}, nil
}
func (c *mockClient) CreateTopic(_ domain.CreateTopicRequest) error { return nil }
func (c *mockClient) DeleteTopic(_ string) error                    { return nil }
func (c *mockClient) UpdateTopicConfig(_ string, _ domain.UpdateTopicConfigRequest) error {
	return nil
}
func (c *mockClient) IncreasePartitions(_ string, _ domain.IncreasePartitionsRequest) error {
	return nil
}
func (c *mockClient) StreamMessages(_ context.Context, _ string, _ chan<- domain.Message) {}
func (c *mockClient) WriteMessage(_ context.Context, _ string, _ domain.Message)          {}
func (c *mockClient) Close()                                                              {}

type testFactory struct {
	topicsPerCluster map[string]map[string]int
}

func (f testFactory) CreateClient(cfg config.ClusterConfig) (domain.KafkaClient, error) {
	topics := map[string]int{}
	if f.topicsPerCluster != nil {
		if t, ok := f.topicsPerCluster[cfg.Name]; ok {
			topics = t
		}
	}
	return &mockClient{healthy: true, topics: topics}, nil
}

// buildServer builds a Server instance for tests with a temporary config file
func buildServer(t *testing.T) *Server {
	t.Helper()
	return buildServerWithFactory(t, testFactory{})
}

// buildServerWithFactory builds a Server with a custom factory for test data injection
func buildServerWithFactory(t *testing.T, factory testFactory) *Server {
	t.Helper()
	tdir := t.TempDir()
	cfgPath := tdir + "/config.yml"
	repo := repository.NewClusterRepository(cfgPath, factory)
	clusterSvc := application.NewClusterService(repo)
	topicSvc := application.NewTopicService(clusterSvc)
	utils.InitLogger()
	return New(clusterSvc, topicSvc)
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

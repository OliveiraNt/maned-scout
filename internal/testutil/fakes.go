package testutil

import (
	"context"

	"github.com/OliveiraNt/maned-scout/internal/config"
	"github.com/OliveiraNt/maned-scout/internal/domain"
	"github.com/twmb/franz-go/pkg/kadm"
)

// FakeKafkaClient is a test double implementing domain.KafkaClient with configurable responses.
type FakeKafkaClient struct {
	Topics         map[string]int
	TopicDetail    *domain.TopicDetail
	Cluster        *domain.Cluster
	Stats          *domain.ClusterStats
	Brokers        []domain.BrokerDetail
	ConsumerGroups []domain.ConsumerGroupSummary
	Lags           kadm.DescribedGroupLags
	Healthy        bool
	Err            error
}

func NewFakeKafkaClient() *FakeKafkaClient {
	return &FakeKafkaClient{Healthy: true, Topics: map[string]int{}}
}

func (f *FakeKafkaClient) IsHealthy() bool                                  { return f.Healthy }
func (f *FakeKafkaClient) ListTopics(_ bool) (map[string]int, error)        { return f.Topics, f.Err }
func (f *FakeKafkaClient) GetClusterInfo() (*domain.Cluster, error)         { return f.Cluster, f.Err }
func (f *FakeKafkaClient) GetClusterStats() (*domain.ClusterStats, error)   { return f.Stats, f.Err }
func (f *FakeKafkaClient) GetBrokerDetails() ([]domain.BrokerDetail, error) { return f.Brokers, f.Err }
func (f *FakeKafkaClient) ListConsumerGroups() ([]domain.ConsumerGroupSummary, error) {
	return f.ConsumerGroups, f.Err
}
func (f *FakeKafkaClient) ListConsumerGroupsWithLagFromTopic(_ context.Context, _ []string, _ string) (kadm.DescribedGroupLags, error) {
	return f.Lags, f.Err
}
func (f *FakeKafkaClient) GetTopicDetail(_ string) (*domain.TopicDetail, error) {
	return f.TopicDetail, f.Err
}
func (f *FakeKafkaClient) CreateTopic(_ domain.CreateTopicRequest) error { return f.Err }
func (f *FakeKafkaClient) DeleteTopic(_ string) error                    { return f.Err }
func (f *FakeKafkaClient) UpdateTopicConfig(_ string, _ domain.UpdateTopicConfigRequest) error {
	return f.Err
}
func (f *FakeKafkaClient) IncreasePartitions(_ string, _ domain.IncreasePartitionsRequest) error {
	return f.Err
}
func (f *FakeKafkaClient) StreamMessages(_ context.Context, _ string, _ chan<- domain.Message) {}
func (f *FakeKafkaClient) WriteMessage(_ context.Context, _ string, _ domain.Message)          {}
func (f *FakeKafkaClient) Close()                                                              {}

// FakeClusterRepository is a simple in-memory repository for tests.
type FakeClusterRepository struct {
	Cfgs    []config.ClusterConfig
	Clients map[string]domain.KafkaClient
}

func NewFakeClusterRepository() *FakeClusterRepository {
	return &FakeClusterRepository{Clients: map[string]domain.KafkaClient{}}
}

func (r *FakeClusterRepository) Save(cfg config.ClusterConfig) error {
	r.Cfgs = append(r.Cfgs, cfg)
	return nil
}
func (r *FakeClusterRepository) Delete(name string) error { delete(r.Clients, name); return nil }
func (r *FakeClusterRepository) FindByName(name string) (config.ClusterConfig, bool) {
	for _, c := range r.Cfgs {
		if c.Name == name {
			return c, true
		}
	}
	return config.ClusterConfig{}, false
}
func (r *FakeClusterRepository) FindAll() []config.ClusterConfig {
	return append([]config.ClusterConfig(nil), r.Cfgs...)
}
func (r *FakeClusterRepository) Watch() error { return nil }
func (r *FakeClusterRepository) GetClient(name string) (domain.KafkaClient, bool) {
	c, ok := r.Clients[name]
	return c, ok
}

// FakeFactory returns a FakeKafkaClient for any config.
type FakeFactory struct {
	Client domain.KafkaClient
	Err    error
}

func (f *FakeFactory) CreateClient(_ config.ClusterConfig) (domain.KafkaClient, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	if f.Client != nil {
		return f.Client, nil
	}
	return NewFakeKafkaClient(), nil
}

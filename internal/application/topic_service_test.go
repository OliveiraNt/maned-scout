package application

import (
	"testing"

	"github.com/OliveiraNt/maned-scout/internal/config"
	"github.com/OliveiraNt/maned-scout/internal/domain"
	"github.com/OliveiraNt/maned-scout/internal/testutil"
	"github.com/OliveiraNt/maned-scout/internal/utils"
	"github.com/stretchr/testify/require"
)

func TestTopicService_ListTopics(t *testing.T) {
	t.Parallel()
	utils.InitLogger()
	repo := testutil.NewFakeClusterRepository()
	repo.Cfgs = []config.ClusterConfig{{Name: "c1", Brokers: []string{"b1"}}}
	repo.Clients["c1"] = &testutil.FakeKafkaClient{Topics: map[string]int{"t": 1}}

	cs := NewClusterService(repo)
	svc := NewTopicService(cs)

	// cluster not found
	_, err := svc.ListTopics("unknown", true)
	require.Error(t, err)

	// client not found
	delete(repo.Clients, "c1")
	_, err = svc.ListTopics("c1", true)
	require.NoError(t, err)

	// success
	repo.Clients["c1"] = &testutil.FakeKafkaClient{Topics: map[string]int{"t": 1}}
	topics, err := svc.ListTopics("c1", false)
	require.NoError(t, err)
	require.Equal(t, 1, topics["t"])
}

func TestTopicService_GetTopicDetailAndMutations(t *testing.T) {
	t.Parallel()
	utils.InitLogger()
	repo := testutil.NewFakeClusterRepository()
	repo.Cfgs = []config.ClusterConfig{{Name: "c1", Brokers: []string{"b1"}}}
	fake := testutil.NewFakeKafkaClient()
	fake.TopicDetail = &domain.TopicDetail{Name: "t"}
	repo.Clients["c1"] = fake

	cs := NewClusterService(repo)
	svc := NewTopicService(cs)

	// missing cluster
	_, err := svc.GetTopicDetail("unknown", "t")
	require.Error(t, err)

	// missing client
	delete(repo.Clients, "c1")
	_, err = svc.GetTopicDetail("c1", "t")
	require.Error(t, err)

	// success
	repo.Clients["c1"] = fake
	detail, err := svc.GetTopicDetail("c1", "t")
	require.NoError(t, err)
	require.Equal(t, "t", detail.Name)

	// create topic validations
	err = svc.CreateTopic("c1", domain.CreateTopicRequest{Name: ""})
	require.Error(t, err)
	err = svc.CreateTopic("unknown", domain.CreateTopicRequest{Name: "t", NumPartitions: 1, ReplicationFactor: 1})
	require.Error(t, err)
	err = svc.CreateTopic("c1", domain.CreateTopicRequest{Name: "t", NumPartitions: 1, ReplicationFactor: 1})
	require.NoError(t, err)

	// update config validation
	err = svc.UpdateTopicConfig("c1", "t", domain.UpdateTopicConfigRequest{})
	require.Error(t, err)
	err = svc.UpdateTopicConfig("c1", "t", domain.UpdateTopicConfigRequest{Configs: map[string]*string{"k": nil}})
	require.NoError(t, err)

	// increase partitions validation
	err = svc.IncreasePartitions("c1", "t", domain.IncreasePartitionsRequest{TotalPartitions: 0})
	require.Error(t, err)
	err = svc.IncreasePartitions("c1", "t", domain.IncreasePartitionsRequest{TotalPartitions: 2})
	require.NoError(t, err)

	// delete topic
	err = svc.DeleteTopic("unknown", "t")
	require.Error(t, err)
	err = svc.DeleteTopic("c1", "t")
	require.NoError(t, err)
}

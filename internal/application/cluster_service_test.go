package application

import (
	"testing"

	"github.com/OliveiraNt/maned-scout/internal/config"
	"github.com/OliveiraNt/maned-scout/internal/domain"
	"github.com/OliveiraNt/maned-scout/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestClusterService_ListAndGetAndAdd(t *testing.T) {
	t.Parallel()
	repo := testutil.NewFakeClusterRepository()
	repo.Cfgs = []config.ClusterConfig{{Name: "c1", Brokers: []string{"b1"}}}
	svc := NewClusterService(repo)

	list := svc.ListClusters()
	require.Len(t, list, 1)

	got, ok := svc.GetCluster("c1")
	require.True(t, ok)
	require.Equal(t, "c1", got.Name)

	// add invalid
	err := svc.AddCluster(config.ClusterConfig{})
	require.Error(t, err)

	// add valid
	err = svc.AddCluster(config.ClusterConfig{Name: "c2", Brokers: []string{"b2"}})
	require.NoError(t, err)
}

func TestClusterService_GetClusterInfo(t *testing.T) {
	t.Parallel()
	repo := testutil.NewFakeClusterRepository()
	cfg := config.ClusterConfig{Name: "c1", Brokers: []string{"b1"}}
	repo.Cfgs = []config.ClusterConfig{cfg}
	repo.Clients["c1"] = &testutil.FakeKafkaClient{Healthy: true, Stats: &domain.ClusterStats{TotalTopics: 1}}

	svc := NewClusterService(repo)
	cluster, stats, err := svc.GetClusterInfo("c1")
	require.NoError(t, err)
	require.NotNil(t, cluster)
	require.True(t, cluster.IsOnline)
	require.Equal(t, 1, stats.TotalTopics)
}

func TestClusterService_GetClusterDetail_Offline(t *testing.T) {
	t.Parallel()
	repo := testutil.NewFakeClusterRepository()
	cfg := config.ClusterConfig{Name: "c1", Brokers: []string{"b1"}}
	repo.Cfgs = []config.ClusterConfig{cfg}
	repo.Clients["c1"] = &testutil.FakeKafkaClient{Healthy: false}

	svc := NewClusterService(repo)
	cluster, topics, stats, brokers, cgs, err := svc.GetClusterDetail("c1")
	require.NoError(t, err)
	require.NotNil(t, cluster)
	require.False(t, cluster.IsOnline)
	require.Empty(t, topics)
	require.Nil(t, stats)
	require.Empty(t, brokers)
	require.Empty(t, cgs)
}

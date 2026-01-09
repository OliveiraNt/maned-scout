package application

import (
	"context"
	"testing"

	"github.com/OliveiraNt/maned-scout/internal/config"
	"github.com/OliveiraNt/maned-scout/internal/testutil"
	"github.com/OliveiraNt/maned-scout/internal/utils"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kadm"
)

func TestConsumerGroupsService_ListConsumerGroupsWithLagFromTopic(t *testing.T) {
	t.Parallel()
	utils.InitLogger()
	repo := testutil.NewFakeClusterRepository()
	repo.Cfgs = []config.ClusterConfig{{Name: "c1", Brokers: []string{"b1"}}}

	// success path uses a fake with known lag map
	lagMap := kadm.DescribedGroupLags{"g1": kadm.DescribedGroupLag{Group: "g1"}}
	repo.Clients["c1"] = &testutil.FakeKafkaClient{Lags: lagMap}

	cs := NewClusterService(repo)
	svc := NewConsumerGroupsService(cs)

	// cluster not found
	_, err := svc.ListConsumerGroupsWithLagFromTopic(context.Background(), "unknown", "")
	require.Error(t, err)

	// client not found
	delete(repo.Clients, "c1")
	_, err = svc.ListConsumerGroupsWithLagFromTopic(context.Background(), "c1", "topic")
	require.Error(t, err)

	// success
	repo.Clients["c1"] = &testutil.FakeKafkaClient{Lags: lagMap}
	lags, err := svc.ListConsumerGroupsWithLagFromTopic(context.Background(), "c1", "topic")
	require.NoError(t, err)
	require.Equal(t, lagMap, lags)
}

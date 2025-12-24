package domain_test

import (
	"context"
	"testing"

	"github.com/OliveiraNt/maned-scout/internal/domain"
	"github.com/OliveiraNt/maned-scout/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestKafkaClientInterface_MethodsReturnDefault(t *testing.T) {
	t.Parallel()
	fc := testutil.NewFakeKafkaClient()
	var client domain.KafkaClient = fc
	require.True(t, client.IsHealthy())

	topics, err := client.ListTopics(false)
	require.NoError(t, err)
	require.NotNil(t, topics)

	_, err = client.GetClusterInfo()
	require.NoError(t, err)

	_, err = client.GetClusterStats()
	require.NoError(t, err)

	_, err = client.GetBrokerDetails()
	require.NoError(t, err)

	_, err = client.ListConsumerGroups()
	require.NoError(t, err)

	_, err = client.ListConsumerGroupsWithLagFromTopic(context.Background(), nil, "")
	require.NoError(t, err)

	_, err = client.GetTopicDetail("t")
	require.NoError(t, err)

	require.NoError(t, client.CreateTopic(domain.CreateTopicRequest{Name: "t"}))
	require.NoError(t, client.DeleteTopic("t"))
	require.NoError(t, client.UpdateTopicConfig("t", domain.UpdateTopicConfigRequest{Configs: map[string]*string{"k": nil}}))
	require.NoError(t, client.IncreasePartitions("t", domain.IncreasePartitionsRequest{TotalPartitions: 1}))

	ch := make(chan domain.Message, 1)
	client.StreamMessages(context.Background(), "t", ch)
	client.WriteMessage(context.Background(), "t", domain.Message{})
	client.Close()
}

package domain_test

import (
	"testing"

	"github.com/OliveiraNt/maned-scout/internal/config"
	"github.com/OliveiraNt/maned-scout/internal/domain"
	"github.com/OliveiraNt/maned-scout/internal/testutil"
	"github.com/stretchr/testify/require"
)

// fakeClient ensures the KafkaClient interface is satisfied; delegates to shared fake for simplicity.
type fakeClient struct{ *testutil.FakeKafkaClient }

func newFakeClient() *fakeClient { return &fakeClient{testutil.NewFakeKafkaClient()} }

func TestClientInterface_Exists(t *testing.T) {
	t.Parallel()
	var c domain.KafkaClient = newFakeClient()
	require.NotNil(t, c)

	cfg := config.ClusterConfig{Name: "local", Brokers: []string{"localhost:9092"}}
	require.Equal(t, "local", cfg.Name)
}

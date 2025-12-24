package domain_test

import (
	"testing"

	"github.com/OliveiraNt/maned-scout/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestCluster_Basic(t *testing.T) {
	t.Parallel()
	c := domain.Cluster{ID: "id", Name: "name", Brokers: []string{"b1"}, IsOnline: true, AuthType: "PLAINTEXT"}
	require.Equal(t, "name", c.Name)
	require.True(t, c.IsOnline)
}

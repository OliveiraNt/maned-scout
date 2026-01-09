package repository_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/OliveiraNt/maned-scout/internal/config"
	"github.com/OliveiraNt/maned-scout/internal/infrastructure/repository"
	"github.com/OliveiraNt/maned-scout/internal/testutil"
	"github.com/OliveiraNt/maned-scout/internal/utils"
	"github.com/stretchr/testify/require"
)

func TestClusterRepository_SaveFindDelete(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yml")
	// initialize empty config file
	require.NoError(t, os.WriteFile(cfgPath, []byte("clusters: []\n"), 0644))

	r := repository.NewClusterRepository(cfgPath, &testutil.FakeFactory{Client: testutil.NewFakeKafkaClient()})

	cfg := config.ClusterConfig{Name: "c1", Brokers: []string{"b1"}}
	require.NoError(t, r.Save(cfg))

	got, ok := r.FindByName("c1")
	require.True(t, ok)
	require.Equal(t, cfg.Name, got.Name)

	list := r.FindAll()
	require.Len(t, list, 1)

	require.NoError(t, r.Delete("c1"))
	_, ok = r.FindByName("c1")
	require.False(t, ok)
}

func TestClusterRepository_LoadFromFile(t *testing.T) {
	t.Parallel()
	utils.InitLogger()
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yml")
	content := "clusters:\n- name: c1\n  brokers:\n  - b1\n"
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0644))

	r := repository.NewClusterRepository(cfgPath, &testutil.FakeFactory{Client: testutil.NewFakeKafkaClient()})
	require.NoError(t, r.LoadFromFile())

	got, ok := r.FindByName("c1")
	require.True(t, ok)
	require.Equal(t, "c1", got.Name)
}

package admin

import (
	"context"
	"strconv"
	"time"

	"github.com/OliveiraNt/kdash/internal/core"
	"github.com/twmb/franz-go/pkg/kadm"
)

// AdminClient defines the minimal interface used by admin helpers.
type AdminClient interface {
	ListTopics(ctx context.Context, topics ...string) (kadm.TopicDetails, error)
	BrokerMetadata(ctx context.Context) (kadm.Metadata, error)
}

// KadmAdmin adapts *kadm.Client to AdminClient interface.
type KadmAdmin struct {
	c *kadm.Client
}

func NewKadmAdmin(c *kadm.Client) *KadmAdmin {
	return &KadmAdmin{c: c}
}

func (k *KadmAdmin) ListTopics(ctx context.Context, topics ...string) (kadm.TopicDetails, error) {
	return k.c.ListTopics(ctx, topics...)
}

func (k *KadmAdmin) BrokerMetadata(ctx context.Context) (kadm.Metadata, error) {
	return k.c.BrokerMetadata(ctx)
}

// ListTopics returns non-internal topics simplified map name->partitions.
func ListTopics(ctx context.Context, admin AdminClient) (map[string]int, error) {
	// use a reasonable timeout
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	m, err := admin.ListTopics(cctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]int)
	for name, info := range m {
		if info.IsInternal {
			continue
		}
		out[name] = len(info.Partitions)
	}
	return out, nil
}

// GetClusterInfo returns a core.Cluster with broker addresses and cluster ID.
func GetClusterInfo(ctx context.Context, admin AdminClient) (*core.Cluster, error) {
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	meta, err := admin.BrokerMetadata(cctx)
	if err != nil {
		return nil, err
	}
	brokers := make([]string, len(meta.Brokers))
	for i, b := range meta.Brokers {
		brokers[i] = b.Host + ":" + strconv.Itoa(int(b.Port))
	}
	return &core.Cluster{ID: meta.Cluster, Brokers: brokers}, nil
}

package admin

import (
	"context"
	"testing"

	"github.com/twmb/franz-go/pkg/kadm"
)

// fakeAdmin implements AdminClient for tests.
type fakeAdmin struct{}

func (f *fakeAdmin) ListGroups(ctx context.Context, groups ...string) (kadm.DescribedGroups, error) {
	//TODO implement me
	panic("implement me")
}

func (f *fakeAdmin) ListTopics(ctx context.Context, topics ...string) (kadm.TopicDetails, error) {
	return kadm.TopicDetails{
		"topic1": {IsInternal: false, Partitions: map[int32]kadm.PartitionDetail{0: {}}},
	}, nil
}

func (f *fakeAdmin) ListTopicsWithInternal(ctx context.Context, topics ...string) (kadm.TopicDetails, error) {
	return kadm.TopicDetails{
		"topic1":     {IsInternal: false, Partitions: map[int32]kadm.PartitionDetail{0: {}}},
		"__internal": {IsInternal: true, Partitions: map[int32]kadm.PartitionDetail{0: {}, 1: {}}},
	}, nil
}

func (f *fakeAdmin) BrokerMetadata(ctx context.Context) (kadm.Metadata, error) {
	return kadm.Metadata{
		Cluster: "cluster-1",
		Brokers: kadm.BrokerDetails{{Host: "broker1", Port: 9092}, {Host: "broker2", Port: 9093}},
	}, nil
}

func TestListTopics(t *testing.T) {
	out, err := ListTopics(context.Background(), &fakeAdmin{}, false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 non-internal topic, got %d", len(out))
	}
	if out["topic1"] != 1 {
		t.Fatalf("expected topic1 partitions 1, got %d", out["topic1"])
	}
}

func TestListTopicsWithInternal(t *testing.T) {
	out, err := ListTopics(context.Background(), &fakeAdmin{}, true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 topics (including internal), got %d", len(out))
	}
	if out["topic1"] != 1 {
		t.Fatalf("expected topic1 partitions 1, got %d", out["topic1"])
	}
	if out["__internal"] != 2 {
		t.Fatalf("expected __internal partitions 2, got %d", out["__internal"])
	}
}

func TestGetClusterInfo(t *testing.T) {
	c, err := GetClusterInfo(context.Background(), &fakeAdmin{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if c.ID != "cluster-1" {
		t.Fatalf("expected cluster id cluster-1, got %s", c.ID)
	}
	if len(c.Brokers) != 2 {
		t.Fatalf("expected 2 brokers, got %d", len(c.Brokers))
	}
}

package domain

import "time"

// Topic represents a Kafka topic with its metadata
type Topic struct {
	Name          string
	Partitions    int
	Replication   int
	TotalMessages int64
	Size          int64
	RetentionMs   int64
	CleanupPolicy string
}

// TopicDetail represents detailed topic information including all configurations
type TopicDetail struct {
	Name              string
	Partitions        int
	ReplicationFactor int
	Configs           map[string]string
	PartitionDetails  []PartitionDetail
}

// PartitionDetail represents detailed partition information
type PartitionDetail struct {
	Partition int32
	Leader    int32
	Replicas  []int32
	ISR       []int32
	Offline   bool
}

// CreateTopicRequest represents a request to create a new topic
type CreateTopicRequest struct {
	Name              string
	NumPartitions     int32
	ReplicationFactor int16
	Configs           map[string]*string
}

// UpdateTopicConfigRequest represents a request to update topic configurations
type UpdateTopicConfigRequest struct {
	Configs map[string]*string
}

// IncreasePartitionsRequest represents a request to increase topic partitions
type IncreasePartitionsRequest struct {
	TotalPartitions int32
}

// Message represents a single message in a Kafka topic, containing key, value, partition, offset, and timestamp information.
type Message struct {
	Key       []byte
	Value     []byte
	Partition int32
	Offset    int64
	Timestamp time.Time
}

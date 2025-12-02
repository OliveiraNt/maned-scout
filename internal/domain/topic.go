package domain

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

package application

import "errors"

// Application-level error constants for common use cases
var (
	ErrClusterNotFound          = errors.New("cluster not found")
	ErrInvalidClusterConfig     = errors.New("invalid cluster configuration")
	ErrInvalidTopicName         = errors.New("topic name is required")
	ErrInvalidPartitionCount    = errors.New("partition count must be greater than 0")
	ErrInvalidReplicationFactor = errors.New("replication factor must be greater than 0")
	ErrInvalidTopicConfig       = errors.New("topic configs are required")
)

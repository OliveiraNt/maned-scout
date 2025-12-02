package application

import "errors"

var (
	// ErrClusterNotFound is returned when a cluster is not found
	ErrClusterNotFound = errors.New("cluster not found")

	// ErrInvalidClusterConfig is returned when cluster configuration is invalid
	ErrInvalidClusterConfig = errors.New("invalid cluster configuration")

	// ErrClusterOffline is returned when a cluster is not reachable
	ErrClusterOffline = errors.New("cluster is offline")
)

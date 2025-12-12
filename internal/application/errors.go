package application

import "errors"

var (
	ErrClusterNotFound      = errors.New("cluster not found")
	ErrInvalidClusterConfig = errors.New("invalid cluster configuration")
	ErrClusterOffline       = errors.New("cluster is offline")
)

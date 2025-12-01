package registry

import (
	"testing"

	"github.com/OliveiraNt/kdash/internal/config"
)

func TestClusterConfigEqual(t *testing.T) {
	a := config.ClusterConfig{
		Name:     "c1",
		Brokers:  []string{"b1:9092"},
		ClientID: "cid",
		TLS:      &config.TLSConfig{Enabled: true, CAFile: "ca.pem"},
		SASL:     &config.SASLConfig{Mechanism: "PLAIN", Username: "u", Password: "p"},
		AWS:      &config.AWSConfig{IAM: false},
		Options:  map[string]string{"k": "v"},
	}
	b := a
	if !clusterConfigEqual(a, b) {
		t.Fatalf("expected equal configs")
	}
	// change brokers
	b.Brokers = []string{"b2:9092"}
	if clusterConfigEqual(a, b) {
		t.Fatalf("expected different configs when brokers differ")
	}
}

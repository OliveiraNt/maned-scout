package kafka

import (
	"os"
	"testing"

	"github.com/OliveiraNt/kdash/internal/config"
)

func TestBuildSASLMechanismPlainAndScram(t *testing.T) {
	// Plain
	s := &config.SASLConfig{Mechanism: "PLAIN", Username: "user", Password: "pass"}
	mech, err := buildSASLMechanism(s)
	if err != nil {
		t.Fatalf("plain build error: %v", err)
	}
	if mech == nil {
		t.Fatalf("expected non-nil mechanism for PLAIN")
	}
	if mech.Name() != "PLAIN" {
		t.Fatalf("expected PLAIN name, got %s", mech.Name())
	}

	// SCRAM-SHA-256
	s2 := &config.SASLConfig{Mechanism: "SCRAM-SHA-256", Username: "user2", Password: "pass2"}
	mech2, err := buildSASLMechanism(s2)
	if err != nil {
		t.Fatalf("scram build error: %v", err)
	}
	if mech2 == nil {
		t.Fatalf("expected non-nil mechanism for SCRAM-SHA-256")
	}
	if mech2.Name() != "SCRAM-SHA-256" {
		t.Fatalf("expected SCRAM-SHA-256 name, got %s", mech2.Name())
	}

	// SCRAM-SHA-512
	s3 := &config.SASLConfig{Mechanism: "SCRAM-SHA-512", Username: "user3", Password: "pass3"}
	mech3, err := buildSASLMechanism(s3)
	if err != nil {
		t.Fatalf("scram512 build error: %v", err)
	}
	if mech3 == nil {
		t.Fatalf("expected non-nil mechanism for SCRAM-SHA-512")
	}
	if mech3.Name() != "SCRAM-SHA-512" {
		t.Fatalf("expected SCRAM-SHA-512 name, got %s", mech3.Name())
	}
}

func TestBuildAWSMechanismWithEnv(t *testing.T) {
	// ensure env is used
	os.Setenv("TEST_AWS_ACCESS_KEY_ID", "AKIA_TEST")
	os.Setenv("TEST_AWS_SECRET_ACCESS_KEY", "SECRET_TEST")
	defer func() {
		os.Unsetenv("TEST_AWS_ACCESS_KEY_ID")
		os.Unsetenv("TEST_AWS_SECRET_ACCESS_KEY")
	}()
	a := &config.AWSConfig{IAM: true, AccessKeyEnv: "TEST_AWS_ACCESS_KEY_ID", SecretKeyEnv: "TEST_AWS_SECRET_ACCESS_KEY"}
	mech, err := buildAWSMechanism(a)
	if err != nil {
		t.Fatalf("aws mech error: %v", err)
	}
	if mech == nil {
		t.Fatalf("expected AWS mechanism when creds are set in env")
	}
	if mech.Name() != "AWS_MSK_IAM" {
		t.Fatalf("expected AWS_MSK_IAM, got %s", mech.Name())
	}
}

func TestBuildTLSConfigBasic(t *testing.T) {
	// basic check: enabled true but no files, set InsecureSkipVerify and ensure returned cfg not nil
	tlsCfg, err := buildTLSConfig(&config.TLSConfig{Enabled: true, InsecureSkipVerify: true})
	if err != nil {
		t.Fatalf("buildTLSConfig error: %v", err)
	}
	if tlsCfg == nil {
		t.Fatalf("expected non-nil tls.Config")
	}
	if !tlsCfg.InsecureSkipVerify {
		t.Fatalf("expected InsecureSkipVerify true")
	}
}

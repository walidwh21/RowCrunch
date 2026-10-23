package settings

import (
	"io"
	"testing"
	"time"
)

func TestDefaults(t *testing.T) {
	for _, k := range []string{"CRUNCH_INPUT", "CRUNCH_MODE", "CRUNCH_WORKERS", "CRUNCH_TIMEOUT"} {
		t.Setenv(k, "")
	}
	s, err := Load(nil, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if s.Mode != ModeParallel || s.Workers < 1 || s.Timeout != 10*time.Minute {
		t.Fatalf("unexpected defaults: %+v", s)
	}
}

func TestEnvThenFlagPrecedence(t *testing.T) {
	t.Setenv("CRUNCH_WORKERS", "3")
	t.Setenv("CRUNCH_MODE", "sequential")

	s, err := Load(nil, io.Discard)
	if err != nil || s.Workers != 3 || s.Mode != ModeSequential {
		t.Fatalf("env not honoured: %+v, %v", s, err)
	}

	s, err = Load([]string{"-workers", "5", "-mode", "parallel"}, io.Discard)
	if err != nil || s.Workers != 5 || s.Mode != ModeParallel {
		t.Fatalf("flags must win over env: %+v, %v", s, err)
	}
}

func TestValidation(t *testing.T) {
	if _, err := Load([]string{"-mode", "turbo"}, io.Discard); err == nil {
		t.Error("unknown mode should be rejected")
	}
	if _, err := Load([]string{"-workers", "0"}, io.Discard); err == nil {
		t.Error("zero workers should be rejected")
	}
}

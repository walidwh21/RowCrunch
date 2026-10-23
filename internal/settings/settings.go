// Package settings resolves the runtime configuration. Every option can be
// given as a command-line flag; its default comes from an environment
// variable, which in turn falls back to a built-in value.
package settings

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"time"
)

// Execution modes.
const (
	ModeSequential = "sequential"
	ModeParallel   = "parallel"
)

// Settings is the resolved configuration of one run.
type Settings struct {
	Input    string
	Output   string
	Metrics  string
	Mode     string
	Workers  int
	Timeout  time.Duration
	MemoryMB uint64
}

// Load parses args (without the program name).
func Load(args []string, errOut io.Writer) (Settings, error) {
	var s Settings

	fs := flag.NewFlagSet("crunch", flag.ContinueOnError)
	fs.SetOutput(errOut)
	fs.StringVar(&s.Input, "input", envString("CRUNCH_INPUT", "data/measurements.csv"),
		"CSV file with `station;value` lines [CRUNCH_INPUT]")
	fs.StringVar(&s.Output, "output", envString("CRUNCH_OUTPUT", "data/report.csv"),
		"where to write the aggregated report [CRUNCH_OUTPUT]")
	fs.StringVar(&s.Metrics, "metrics", envString("CRUNCH_METRICS", "data/runs.log"),
		"file that receives one line per run (empty disables) [CRUNCH_METRICS]")
	fs.StringVar(&s.Mode, "mode", envString("CRUNCH_MODE", ModeParallel),
		"sequential | parallel [CRUNCH_MODE]")
	fs.IntVar(&s.Workers, "workers", envInt("CRUNCH_WORKERS", runtime.NumCPU()),
		"number of concurrent readers in parallel mode [CRUNCH_WORKERS]")
	fs.DurationVar(&s.Timeout, "timeout", envDuration("CRUNCH_TIMEOUT", 10*time.Minute),
		"abort when the run takes longer (0 disables) [CRUNCH_TIMEOUT]")
	fs.Uint64Var(&s.MemoryMB, "mem-limit", uint64(envInt("CRUNCH_MEM_LIMIT_MB", 24*1024)),
		"abort when the process holds more than this many MiB (0 disables) [CRUNCH_MEM_LIMIT_MB]")

	if err := fs.Parse(args); err != nil {
		return Settings{}, err
	}
	return s, s.validate()
}

func (s Settings) validate() error {
	switch s.Mode {
	case ModeSequential, ModeParallel:
	default:
		return fmt.Errorf("unknown mode %q (want %s or %s)", s.Mode, ModeSequential, ModeParallel)
	}
	if s.Workers < 1 {
		return fmt.Errorf("workers must be at least 1, got %d", s.Workers)
	}
	if s.Input == "" || s.Output == "" {
		return fmt.Errorf("input and output paths must not be empty")
	}
	return nil
}

func envString(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

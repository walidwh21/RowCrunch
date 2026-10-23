// Command crunch aggregates a huge `station;value` CSV into per-station
// min / max / mean / count and writes the sorted result to a file.
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"rowcrunch/internal/engine"
	"rowcrunch/internal/probe"
	"rowcrunch/internal/readings"
	"rowcrunch/internal/settings"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cfg, err := settings.Load(args, os.Stderr)
	if err != nil {
		return err
	}

	info, err := os.Stat(cfg.Input)
	if err != nil {
		return err
	}
	probe.PrintHost(os.Stdout)
	fmt.Printf("input   : %s (%.2f GiB)\n", cfg.Input, float64(info.Size())/(1<<30))
	fmt.Printf("mode    : %s, workers=%d\n", cfg.Mode, cfg.Workers)

	var summary engine.Summary
	limits := probe.Limits{Time: cfg.Timeout, MemoryMB: cfg.MemoryMB}

	rep, err := probe.Run(context.Background(), limits, func(ctx context.Context) error {
		var table *readings.Table
		var err error
		if cfg.Mode == settings.ModeSequential {
			table, summary, err = engine.Sequential(ctx, cfg.Input)
		} else {
			table, summary, err = engine.Parallel(ctx, cfg.Input, cfg.Workers)
		}
		if err != nil {
			return err
		}
		return writeReport(cfg.Output, table)
	})
	if err != nil {
		return err
	}

	fmt.Printf("rows    : %d aggregated, %d skipped\n", summary.Rows, summary.Skipped)
	fmt.Printf("elapsed : %s\n", rep.Elapsed.Round(1e6))
	fmt.Printf("memory  : %.1f MiB allocated over the run, %.1f MiB peak from the OS, %d GC cycles\n",
		float64(rep.TotalAlloc)/(1<<20), float64(rep.PeakSys)/(1<<20), rep.GCCycles)

	if cfg.Metrics != "" {
		label := fmt.Sprintf("mode=%s workers=%d", cfg.Mode, summary.Workers)
		if err := probe.AppendLog(cfg.Metrics, label, rep); err != nil {
			fmt.Fprintln(os.Stderr, "warning: could not write metrics:", err)
		}
	}
	return preview(cfg.Output, 5)
}

func writeReport(path string, table *readings.Table) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := table.WriteReport(f); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// preview prints the first n lines of the report.
func preview(path string, n int) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Printf("report  : %s (first %d lines)\n", path, n)
	sc := bufio.NewScanner(f)
	for i := 0; i < n && sc.Scan(); i++ {
		fmt.Println("  " + sc.Text())
	}
	return sc.Err()
}

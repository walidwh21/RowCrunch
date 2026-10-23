// Package probe measures a run (wall time, allocations, peak memory) and
// enforces the time and memory budgets by cancelling the run's context.
package probe

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"
)

// Errors returned when a budget is exceeded.
var (
	ErrTimeLimit   = errors.New("time limit exceeded")
	ErrMemoryLimit = errors.New("memory limit exceeded")
)

// Limits are the budgets of a run. Zero means "unlimited".
type Limits struct {
	Time     time.Duration
	MemoryMB uint64
}

// Report is what a measured run consumed.
type Report struct {
	Elapsed    time.Duration
	TotalAlloc uint64 // bytes allocated over the run (cumulative)
	PeakSys    uint64 // highest memory obtained from the OS during the run
	HeapInUse  uint64 // heap in use right after the run
	GCCycles   uint32
}

const sampleInterval = 250 * time.Millisecond

// Run executes fn under the given limits and returns what it consumed.
func Run(parent context.Context, lim Limits, fn func(ctx context.Context) error) (Report, error) {
	ctx, cancel := context.WithCancelCause(parent)
	defer cancel(nil)
	if lim.Time > 0 {
		var stop context.CancelFunc
		ctx, stop = context.WithTimeoutCause(ctx, lim.Time, ErrTimeLimit)
		defer stop()
	}

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	peak := make(chan uint64, 1)
	stopWatch := make(chan struct{})
	go watch(cancel, lim, stopWatch, peak)

	start := time.Now()
	err := fn(ctx)
	elapsed := time.Since(start)

	close(stopWatch)
	peakSys := <-peak

	if err != nil {
		if cause := context.Cause(ctx); cause != nil {
			err = cause
		}
	}

	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	if after.Sys > peakSys {
		peakSys = after.Sys
	}

	return Report{
		Elapsed:    elapsed,
		TotalAlloc: after.TotalAlloc - before.TotalAlloc,
		PeakSys:    peakSys,
		HeapInUse:  after.HeapInuse,
		GCCycles:   after.NumGC - before.NumGC,
	}, err
}

// watch samples memory until stop is closed, cancelling the run when the
// memory budget is exceeded. It reports the highest value seen on peak.
func watch(cancel context.CancelCauseFunc, lim Limits,
	stop <-chan struct{}, peak chan<- uint64) {

	var highest uint64
	ticker := time.NewTicker(sampleInterval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			peak <- highest
			return
		case <-ticker.C:
			var ms runtime.MemStats
			runtime.ReadMemStats(&ms)
			if ms.Sys > highest {
				highest = ms.Sys
			}
			if lim.MemoryMB > 0 && ms.Sys > lim.MemoryMB<<20 {
				cancel(ErrMemoryLimit)
			}
		}
	}
}

// PrintHost writes a short description of the machine and runtime.
func PrintHost(w io.Writer) {
	fmt.Fprintf(w, "host    : %s/%s, %d CPUs, GOMAXPROCS=%d, %s\n",
		runtime.GOOS, runtime.GOARCH, runtime.NumCPU(), runtime.GOMAXPROCS(0), runtime.Version())
}

// AppendLog appends one line describing a run to the file at path.
func AppendLog(path, label string, rep Report) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintf(f,
		"%s %s elapsed=%s total_alloc_mb=%.1f peak_sys_mb=%.1f heap_mb=%.1f gc=%d\n",
		time.Now().UTC().Format(time.RFC3339), label,
		rep.Elapsed.Round(time.Millisecond),
		mb(rep.TotalAlloc), mb(rep.PeakSys), mb(rep.HeapInUse), rep.GCCycles)
	return err
}

func mb(b uint64) float64 { return float64(b) / (1 << 20) }

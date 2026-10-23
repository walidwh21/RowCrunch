package probe

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRunReportsUsage(t *testing.T) {
	rep, err := Run(context.Background(), Limits{}, func(ctx context.Context) error {
		time.Sleep(20 * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Elapsed < 20*time.Millisecond {
		t.Fatalf("elapsed %s is shorter than the work", rep.Elapsed)
	}
	if rep.PeakSys == 0 {
		t.Fatal("peak memory was not recorded")
	}
}

func TestRunTimeLimit(t *testing.T) {
	_, err := Run(context.Background(), Limits{Time: 30 * time.Millisecond}, func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})
	if !errors.Is(err, ErrTimeLimit) {
		t.Fatalf("err = %v, want ErrTimeLimit", err)
	}
}

func TestRunKeepsWorkError(t *testing.T) {
	boom := errors.New("boom")
	_, err := Run(context.Background(), Limits{Time: time.Minute}, func(ctx context.Context) error {
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want boom", err)
	}
}

func TestAppendLog(t *testing.T) {
	path := t.TempDir() + "/runs.log"
	rep := Report{Elapsed: 1500 * time.Millisecond, TotalAlloc: 5 << 20, PeakSys: 10 << 20, GCCycles: 3}
	for i := 0; i < 2; i++ {
		if err := AppendLog(path, "mode=parallel workers=4", rep); err != nil {
			t.Fatal(err)
		}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], "elapsed=1.5s") || !strings.Contains(lines[0], "gc=3") {
		t.Fatalf("unexpected log content:\n%s", b)
	}
}

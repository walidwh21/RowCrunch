package engine

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"rowcrunch/internal/readings"
)

const (
	samplePath   = "../../testdata/sample.csv"
	expectedPath = "../../testdata/expected.csv"
	sampleRows   = 5000 // valid readings in the sample
	sampleBad    = 2    // malformed lines in the sample
)

func render(t *testing.T, tb *readings.Table) string {
	t.Helper()
	var sb strings.Builder
	if err := tb.WriteReport(&sb); err != nil {
		t.Fatal(err)
	}
	return sb.String()
}

func golden(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestSequentialMatchesGolden(t *testing.T) {
	tb, sum, err := Sequential(context.Background(), samplePath)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Rows != sampleRows || sum.Skipped != sampleBad {
		t.Fatalf("summary = %+v, want rows=%d skipped=%d", sum, sampleRows, sampleBad)
	}
	if got, want := render(t, tb), golden(t); got != want {
		t.Fatalf("report differs from golden file\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestParallelMatchesGolden(t *testing.T) {
	want := golden(t)
	for _, workers := range []int{1, 2, 3, 7, 16, 200} {
		tb, sum, err := Parallel(context.Background(), samplePath, workers)
		if err != nil {
			t.Fatalf("workers=%d: %v", workers, err)
		}
		if sum.Rows != sampleRows || sum.Skipped != sampleBad {
			t.Fatalf("workers=%d: summary = %+v", workers, sum)
		}
		if got := render(t, tb); got != want {
			t.Fatalf("workers=%d: report differs from golden file", workers)
		}
	}
}

func TestSplitPointsAreLineAligned(t *testing.T) {
	data, err := os.ReadFile(samplePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, parts := range []int{1, 2, 5, 13, 64, 10000} {
		pts, err := SplitPoints(bytes.NewReader(data), int64(len(data)), parts)
		if err != nil {
			t.Fatal(err)
		}
		if pts[0] != 0 || pts[len(pts)-1] != int64(len(data)) {
			t.Fatalf("parts=%d: bad ends %v", parts, pts)
		}
		for i := 1; i < len(pts); i++ {
			if pts[i] <= pts[i-1] {
				t.Fatalf("parts=%d: offsets not increasing: %v", parts, pts)
			}
			if i < len(pts)-1 && data[pts[i]-1] != '\n' {
				t.Fatalf("parts=%d: offset %d does not start a line", parts, pts[i])
			}
		}
	}
}

func TestEmptyFile(t *testing.T) {
	path := t.TempDir() + "/empty.csv"
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, workers := range []int{1, 4} {
		tb, sum, err := Parallel(context.Background(), path, workers)
		if err != nil || tb.Len() != 0 || sum.Rows != 0 {
			t.Fatalf("workers=%d: got (%v, %+v, %v)", workers, tb, sum, err)
		}
	}
}

func TestOversizedLineIsSkipped(t *testing.T) {
	long := strings.Repeat("x", 2*readBufferSize)
	input := "A;1.00\n" + long + ";5.00\nA;3.00\n"
	path := t.TempDir() + "/long.csv"
	if err := os.WriteFile(path, []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}
	tb, sum, err := Sequential(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Rows != 2 || sum.Skipped != 1 {
		t.Fatalf("summary = %+v, want rows=2 skipped=1", sum)
	}
	if agg, ok := tb.Get("A"); !ok || agg.Min != 100 || agg.Max != 300 {
		t.Fatalf("unexpected aggregate: %+v", agg)
	}
}

func TestCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := Sequential(ctx, samplePath); err == nil {
		t.Fatal("expected an error for a cancelled context")
	}
	if _, _, err := Parallel(ctx, samplePath, 4); err == nil {
		t.Fatal("expected an error for a cancelled context")
	}
}

func TestMissingFile(t *testing.T) {
	if _, _, err := Sequential(context.Background(), "does-not-exist.csv"); err == nil {
		t.Fatal("expected an error for a missing file")
	}
}

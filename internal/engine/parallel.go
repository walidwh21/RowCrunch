package engine

import (
	"bytes"
	"context"
	"io"
	"os"
	"sync"

	"rowcrunch/internal/readings"
)

// Parallel splits the file into up to `workers` newline-aligned ranges and
// processes them concurrently. Every worker owns a private table, so the hot
// loop needs no locks; tables are merged once at the end.
func Parallel(ctx context.Context, path string, workers int) (*readings.Table, Summary, error) {
	if workers < 1 {
		workers = 1
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, Summary{}, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, Summary{}, err
	}

	bounds, err := SplitPoints(f, info.Size(), workers)
	if err != nil {
		return nil, Summary{}, err
	}
	ranges := len(bounds) - 1

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type outcome struct {
		table   *readings.Table
		skipped int64
		err     error
	}
	outcomes := make([]outcome, ranges)

	var wg sync.WaitGroup
	for i := 0; i < ranges; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			section := io.NewSectionReader(f, bounds[i], bounds[i+1]-bounds[i])
			table := readings.NewTable(1024)
			skipped, err := scan(ctx, section, table)
			if err != nil {
				cancel() // stop the siblings early
			}
			outcomes[i] = outcome{table: table, skipped: skipped, err: err}
		}(i)
	}
	wg.Wait()

	merged := readings.NewTable(1024)
	var skipped int64
	for _, o := range outcomes {
		if o.err != nil {
			return nil, Summary{}, o.err
		}
		merged.Merge(o.table)
		skipped += o.skipped
	}
	return merged, Summary{Rows: merged.Rows(), Skipped: skipped, Workers: ranges}, nil
}

// SplitPoints returns byte offsets b[0]=0 < b[1] < ... < b[n]=size such that
// every b[i] (0 < i < n) is the first byte of a line. Consecutive offsets
// delimit the ranges handed to the workers. Fewer than `parts` ranges are
// returned for small files.
func SplitPoints(r io.ReaderAt, size int64, parts int) ([]int64, error) {
	points := []int64{0}
	for i := 1; i < parts; i++ {
		guess := size * int64(i) / int64(parts)
		aligned, err := nextLineStart(r, guess, size)
		if err != nil {
			return nil, err
		}
		if aligned > points[len(points)-1] && aligned < size {
			points = append(points, aligned)
		}
	}
	return append(points, size), nil
}

// nextLineStart returns the offset just after the first '\n' at or beyond pos,
// or size when no further newline exists.
func nextLineStart(r io.ReaderAt, pos, size int64) (int64, error) {
	buf := make([]byte, 4096)
	for pos < size {
		n, err := r.ReadAt(buf, pos)
		if i := bytes.IndexByte(buf[:n], '\n'); i >= 0 {
			return pos + int64(i) + 1, nil
		}
		pos += int64(n)
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
	}
	return size, nil
}

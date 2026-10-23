// Package engine turns a CSV file of "station;value" lines into a
// readings.Table. Two strategies share the same line scanner:
//
//   - Sequential: one reader walks the whole file (the baseline).
//   - Parallel:   the file is cut into newline-aligned byte ranges and each
//     range is processed by its own worker with a private table.
package engine

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os"

	"rowcrunch/internal/readings"
)

const (
	readBufferSize = 1 << 20 // 1 MiB per reader
	ctxCheckEvery  = 1 << 16 // lines between cancellation checks
)

// Summary describes what a run saw.
type Summary struct {
	Rows    int64 // readings aggregated
	Skipped int64 // malformed or oversized lines that were ignored
	Workers int   // number of concurrent readers used
}

var newline = []byte{'\n'}

// scan reads r line by line and feeds every valid reading into t.
// It returns the number of lines that had to be skipped.
func scan(ctx context.Context, r io.Reader, t *readings.Table) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	br := bufio.NewReaderSize(r, readBufferSize)

	var skipped int64
	var seen uint
	for {
		line, rerr := br.ReadSlice('\n')

		switch {
		case rerr == nil || rerr == io.EOF:
			line = bytes.TrimSuffix(line, newline)
			if len(line) > 0 {
				if station, v, ok := readings.ParseLine(line); ok {
					t.Observe(station, v)
				} else {
					skipped++
				}
			}
		case rerr == bufio.ErrBufferFull:
			// A single line longer than the buffer cannot be valid input.
			skipped++
			for rerr == bufio.ErrBufferFull {
				_, rerr = br.ReadSlice('\n')
			}
			if rerr != nil && rerr != io.EOF {
				return skipped, rerr
			}
		default:
			return skipped, rerr
		}

		if rerr == io.EOF {
			return skipped, nil
		}

		seen++
		if seen%ctxCheckEvery == 0 {
			if err := ctx.Err(); err != nil {
				return skipped, err
			}
		}
	}
}

// Sequential processes the whole file with a single reader.
func Sequential(ctx context.Context, path string) (*readings.Table, Summary, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, Summary{}, err
	}
	defer f.Close()

	table := readings.NewTable(1024)
	skipped, err := scan(ctx, f, table)
	if err != nil {
		return nil, Summary{}, err
	}
	return table, Summary{Rows: table.Rows(), Skipped: skipped, Workers: 1}, nil
}

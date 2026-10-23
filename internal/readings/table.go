package readings

import (
	"bufio"
	"io"
	"sort"
	"strconv"
)

// Agg accumulates the statistics of one station. Values are hundredths.
type Agg struct {
	Min, Max int32
	Sum      int64
	Count    int64
}

// Add folds one reading into the aggregate.
func (a *Agg) Add(v int32) {
	if v < a.Min {
		a.Min = v
	}
	if v > a.Max {
		a.Max = v
	}
	a.Sum += int64(v)
	a.Count++
}

// Merge folds another aggregate into this one.
func (a *Agg) Merge(o *Agg) {
	if o.Min < a.Min {
		a.Min = o.Min
	}
	if o.Max > a.Max {
		a.Max = o.Max
	}
	a.Sum += o.Sum
	a.Count += o.Count
}

// Mean returns the arithmetic mean in degrees.
func (a *Agg) Mean() float64 {
	return float64(a.Sum) / float64(a.Count) / 100.0
}

// Table maps station names to their aggregate. It is not safe for concurrent
// use; each worker owns one Table and the results are merged afterwards.
type Table struct {
	m map[string]*Agg
}

// NewTable creates a table sized for roughly hint distinct stations.
func NewTable(hint int) *Table {
	return &Table{m: make(map[string]*Agg, hint)}
}

// Observe records one reading. The lookup m[string(b)] does not allocate; a
// string is only materialised the first time a station is seen.
func (t *Table) Observe(station []byte, v int32) {
	if a, ok := t.m[string(station)]; ok {
		a.Add(v)
		return
	}
	t.m[string(station)] = &Agg{Min: v, Max: v, Sum: int64(v), Count: 1}
}

// Merge folds every station of o into t.
func (t *Table) Merge(o *Table) {
	for name, agg := range o.m {
		if mine, ok := t.m[name]; ok {
			mine.Merge(agg)
		} else {
			t.m[name] = agg
		}
	}
}

// Len returns the number of distinct stations.
func (t *Table) Len() int { return len(t.m) }

// Get returns the aggregate for a station, if present.
func (t *Table) Get(name string) (*Agg, bool) {
	a, ok := t.m[name]
	return a, ok
}

// Rows returns the total number of readings in the table.
func (t *Table) Rows() int64 {
	var n int64
	for _, a := range t.m {
		n += a.Count
	}
	return n
}

// WriteReport writes one line per station, sorted by name:
//
//	station;min;max;mean;count
func (t *Table) WriteReport(w io.Writer) error {
	names := make([]string, 0, len(t.m))
	for name := range t.m {
		names = append(names, name)
	}
	sort.Strings(names)

	bw := bufio.NewWriterSize(w, 1<<20)
	line := make([]byte, 0, 128)
	for _, name := range names {
		a := t.m[name]
		line = append(line[:0], name...)
		line = append(line, ';')
		line = AppendHundredths(line, a.Min)
		line = append(line, ';')
		line = AppendHundredths(line, a.Max)
		line = append(line, ';')
		line = strconv.AppendFloat(line, a.Mean(), 'f', 2, 64)
		line = append(line, ';')
		line = strconv.AppendInt(line, a.Count, 10)
		line = append(line, '\n')
		if _, err := bw.Write(line); err != nil {
			return err
		}
	}
	return bw.Flush()
}

// AppendHundredths appends v (in hundredths) formatted with two decimals.
func AppendHundredths(dst []byte, v int32) []byte {
	if v < 0 {
		dst = append(dst, '-')
		v = -v
	}
	dst = strconv.AppendInt(dst, int64(v/100), 10)
	dst = append(dst, '.')
	f := v % 100
	return append(dst, byte('0'+f/10), byte('0'+f%10))
}

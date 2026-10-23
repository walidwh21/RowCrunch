// Package readings holds the data model of the pipeline: parsing a single
// "station;value" line, accumulating statistics per station and rendering the
// final report.
//
// Temperatures are stored as fixed-point integers (hundredths of a degree).
// This avoids float parsing on the hot path and makes aggregation exact.
package readings

import "bytes"

// MaxWholeDigits bounds the integer part so that value*100 always fits int32.
const MaxWholeDigits = 7

// ParseLine splits one input line into station name and value (in hundredths).
// A trailing carriage return is tolerated. The returned station slice aliases
// the input line and is only valid until the caller reuses the buffer.
func ParseLine(line []byte) (station []byte, value int32, ok bool) {
	if n := len(line); n > 0 && line[n-1] == '\r' {
		line = line[:n-1]
	}
	sep := bytes.LastIndexByte(line, ';')
	if sep <= 0 {
		return nil, 0, false
	}
	v, ok := ParseHundredths(line[sep+1:])
	if !ok {
		return nil, 0, false
	}
	return line[:sep], v, true
}

// ParseHundredths converts text such as "-12.5", "+3", "0.07" or "41.239"
// into hundredths. Digits beyond the second fractional digit are truncated.
// It returns ok=false for empty input, stray characters or values that are
// too large.
func ParseHundredths(b []byte) (int32, bool) {
	if len(b) == 0 {
		return 0, false
	}
	i := 0
	negative := false
	switch b[0] {
	case '-':
		negative = true
		i++
	case '+':
		i++
	}

	var whole int32
	wholeDigits := 0
	for ; i < len(b) && b[i] >= '0' && b[i] <= '9'; i++ {
		whole = whole*10 + int32(b[i]-'0')
		wholeDigits++
		if wholeDigits > MaxWholeDigits {
			return 0, false
		}
	}
	if wholeDigits == 0 {
		return 0, false
	}

	var frac int32
	kept, seen := 0, 0
	if i < len(b) && b[i] == '.' {
		i++
		for ; i < len(b) && b[i] >= '0' && b[i] <= '9'; i++ {
			seen++
			if kept < 2 {
				frac = frac*10 + int32(b[i]-'0')
				kept++
			}
		}
		if seen == 0 {
			return 0, false
		}
	}
	if i != len(b) {
		return 0, false
	}
	for ; kept < 2; kept++ {
		frac *= 10
	}

	v := whole*100 + frac
	if negative {
		v = -v
	}
	return v, true
}

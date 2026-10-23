package readings

import (
	"strings"
	"testing"
)

func TestParseHundredths(t *testing.T) {
	cases := []struct {
		in   string
		want int32
		ok   bool
	}{
		{"0", 0, true},
		{"7", 700, true},
		{"7.5", 750, true},
		{"7.05", 705, true},
		{"-12.34", -1234, true},
		{"+3.1", 310, true},
		{"-0.07", -7, true},
		{"41.239", 4123, true}, // extra digits are truncated
		{"9999999.99", 999999999, true},
		{"", 0, false},
		{"-", 0, false},
		{".5", 0, false},
		{"5.", 0, false},
		{"1e3", 0, false},
		{"12a", 0, false},
		{"1.2.3", 0, false},
		{"12345678", 0, false}, // too many integer digits
	}
	for _, c := range cases {
		got, ok := ParseHundredths([]byte(c.in))
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("ParseHundredths(%q) = (%d, %v), want (%d, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestParseLine(t *testing.T) {
	st, v, ok := ParseLine([]byte("São Tomé;-3.5\r"))
	if !ok || string(st) != "São Tomé" || v != -350 {
		t.Fatalf("got (%q, %d, %v)", st, v, ok)
	}

	// The value is taken after the LAST separator.
	st, v, ok = ParseLine([]byte("a;b;2.25"))
	if !ok || string(st) != "a;b" || v != 225 {
		t.Fatalf("got (%q, %d, %v)", st, v, ok)
	}

	for _, bad := range []string{"", "no separator", ";1.0", "x;", "x;abc"} {
		if _, _, ok := ParseLine([]byte(bad)); ok {
			t.Errorf("ParseLine(%q) should fail", bad)
		}
	}
}

func TestTableObserveMergeReport(t *testing.T) {
	a := NewTable(4)
	a.Observe([]byte("B"), 100)
	a.Observe([]byte("A"), -250)
	a.Observe([]byte("B"), 300)

	b := NewTable(4)
	b.Observe([]byte("B"), -50)
	b.Observe([]byte("C"), 5)

	a.Merge(b)

	if a.Len() != 3 || a.Rows() != 5 {
		t.Fatalf("len=%d rows=%d, want 3 and 5", a.Len(), a.Rows())
	}
	agg, ok := a.Get("B")
	if !ok || agg.Min != -50 || agg.Max != 300 || agg.Count != 3 || agg.Sum != 350 {
		t.Fatalf("unexpected aggregate for B: %+v", agg)
	}

	var sb strings.Builder
	if err := a.WriteReport(&sb); err != nil {
		t.Fatal(err)
	}
	want := "A;-2.50;-2.50;-2.50;1\n" +
		"B;-0.50;3.00;1.17;3\n" +
		"C;0.05;0.05;0.05;1\n"
	if sb.String() != want {
		t.Fatalf("report mismatch\n got: %q\nwant: %q", sb.String(), want)
	}
}

func TestAppendHundredths(t *testing.T) {
	cases := map[int32]string{0: "0.00", 5: "0.05", -5: "-0.05", 1234: "12.34", -9999: "-99.99", 700: "7.00"}
	for in, want := range cases {
		if got := string(AppendHundredths(nil, in)); got != want {
			t.Errorf("AppendHundredths(%d) = %q, want %q", in, got, want)
		}
	}
}

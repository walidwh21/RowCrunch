// Command datagen writes a synthetic measurements file for benchmarking.
//
// Every station gets its own base temperature; each reading is that base plus
// Gaussian noise, clamped to the range the parser accepts. A fixed seed makes
// the output reproducible.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"
)

var stations = []string{
	"Albania", "Algeria", "Angola", "Argentina", "Armenia", "Australia", "Austria", "Azerbaijan",
	"Bahamas", "Bahrain", "Bangladesh", "Belarus", "Belgium", "Belize", "Benin", "Bhutan", "Bolivia",
	"Botswana", "Brazil", "Brunei", "Bulgaria", "Burkina Faso", "Burundi", "Cambodia", "Cameroon",
	"Canada", "Cape Verde", "Chad", "Chile", "China", "Colombia", "Comoros", "Costa Rica", "Croatia",
	"Cuba", "Cyprus", "Czech Republic", "Denmark", "Djibouti", "Dominican Republic", "Ecuador",
	"Egypt", "El Salvador", "Eritrea", "Estonia", "Eswatini", "Ethiopia", "Fiji", "Finland", "France",
	"Gabon", "Georgia", "Germany", "Ghana", "Greece", "Greenland", "Guatemala", "Guinea", "Guyana",
	"Haiti", "Honduras", "Hungary", "Iceland", "India", "Indonesia", "Iran", "Iraq", "Ireland",
	"Israel", "Italy", "Ivory Coast", "Jamaica", "Japan", "Jordan", "Kazakhstan", "Kenya", "Kuwait",
	"Kyrgyzstan", "Laos", "Latvia", "Lebanon", "Lesotho", "Liberia", "Libya", "Lithuania",
	"Luxembourg", "Madagascar", "Malawi", "Malaysia", "Maldives", "Mali", "Malta", "Mauritania",
	"Mexico", "Moldova", "Mongolia", "Montenegro", "Morocco", "Mozambique", "Namibia", "Nepal",
	"Netherlands", "New Zealand", "Nicaragua", "Niger", "Nigeria", "Norway", "Oman", "Pakistan",
	"Panama", "Papua New Guinea", "Paraguay", "Peru", "Poland", "Portugal", "Qatar", "Romania",
	"Russia", "Rwanda", "São Tomé and Príncipe", "Saudi Arabia", "Senegal", "Serbia", "Seychelles",
	"Sierra Leone", "Singapore", "Slovakia", "Slovenia", "Somalia", "South Africa", "South Korea",
	"South Sudan", "Spain", "Sri Lanka", "Sudan", "Suriname", "Sweden", "Switzerland", "Syria",
	"Taiwan", "Tajikistan", "Tanzania", "Thailand", "The Gambia", "Togo", "Tunisia", "Turkey",
	"Turkmenistan", "Uganda", "Ukraine", "United Arab Emirates", "United Kingdom", "United States",
	"Uruguay", "Uzbekistan", "Venezuela", "Vietnam", "Yemen", "Zambia", "Zimbabwe",
}

const (
	maxAbs     = 99.99
	noiseSigma = 10.0
)

func main() {
	rows := flag.Int64("rows", 1_000_000_000, "number of lines to write")
	out := flag.String("out", "data/measurements.csv", "output file")
	seed := flag.Int64("seed", 42, "random seed (same seed, same file)")
	flag.Parse()

	if err := generate(*out, *rows, *seed); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func generate(path string, rows, seed int64) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	rng := rand.New(rand.NewSource(seed))
	base := make([]float64, len(stations))
	for i := range base {
		base[i] = -10 + rng.Float64()*45 // between -10 and 35 degrees
	}

	w := bufio.NewWriterSize(f, 4<<20)
	line := make([]byte, 0, 64)
	start := time.Now()

	for n := int64(1); n <= rows; n++ {
		i := rng.Intn(len(stations))
		v := base[i] + rng.NormFloat64()*noiseSigma
		if v > maxAbs {
			v = maxAbs
		} else if v < -maxAbs {
			v = -maxAbs
		}

		line = append(line[:0], stations[i]...)
		line = append(line, ';')
		line = strconv.AppendFloat(line, v, 'f', 2, 64)
		line = append(line, '\n')
		if _, err := w.Write(line); err != nil {
			return err
		}

		if n%50_000_000 == 0 {
			fmt.Printf("%d / %d rows (%s)\n", n, rows, time.Since(start).Round(time.Second))
		}
	}
	return w.Flush()
}

package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"time"
)

type id = string

type pkgid = string

type RawLine struct {
	Action  string    `json:"Action"`
	Package string    `json:"Package"`
	Test    string    `json:"Test"`
	Output  string    `json:"Output"`
	Time    time.Time `json:"Time"`
	Elapsed float64   `json:"Elapsed"`
}

type test struct {
	pkg      pkgid
	name     string
	duration time.Duration
	passed   bool
}

type pkg struct {
	id       pkgid
	duration time.Duration
}

type stats struct {
	packages map[pkgid]*pkg
	tests    map[id]*test
}

func newStats() *stats {
	return &stats{
		packages: make(map[pkgid]*pkg),
		tests:    make(map[id]*test),
	}
}

func (s *stats) testsSortedByDurationDescending() []*test {
	var out []*test
	for _, t := range s.tests {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[j].duration < out[i].duration })
	return out
}

func (s *stats) packagesSortedByDurationDescending() []*pkg {
	var out []*pkg
	for _, p := range s.packages {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[j].duration < out[i].duration })
	return out
}

func testId(pkg pkgid, name string) id {
	return fmt.Sprintf("%s#%s", pkg, name)
}

func readFile(path string) ([]RawLine, error) {
	var lines []RawLine

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	var time0 time.Time

	for scanner.Scan() {
		line := scanner.Text()

		var rawLine RawLine
		err := json.Unmarshal([]byte(line), &rawLine)

		if err != nil {
			return nil, err
		}
		if rawLine.Time.After(time0) {
			lines = append(lines, rawLine)
		}
	}

	return lines, nil
}

func newStatsFromLines(s *stats, lines []RawLine) {
	var time0 time.Time
	for _, line := range lines {
		isValid := line.Time.After(time0) && line.Package != "" && line.Action != ""
		if !isValid {
			continue
		}
		if line.Test != "" {
			t := &test{
				pkg:      line.Package,
				name:     line.Test,
				duration: time.Duration(line.Elapsed * float64(time.Second)),
			}
			switch line.Action {
			case "pass":
				t.passed = true
				s.tests[testId(line.Package, line.Test)] = t
			case "fail":
				t.passed = false
				s.tests[testId(line.Package, line.Test)] = t
			}
		} else {
			p := &pkg{
				id:       line.Package,
				duration: time.Duration(line.Elapsed * float64(time.Second)),
			}
			switch line.Action {
			case "pass":
				s.packages[line.Package] = p
			case "fail":
				s.packages[line.Package] = p
			}
		}
	}
}

func newStatsFromFiles(files []string) *stats {
	s := newStats()
	for _, a := range files {
		lines, err := readFile(a)
		if err != nil {
			log.Fatal(err)
		}
		newStatsFromLines(s, lines)
	}
	return s
}

func main() {
	var statistic string
	flag.StringVar(&statistic, "statistic", "", "Statistic to compute: pkg-time|test-time")
	oldUsage := flag.Usage
	flag.Usage = func() {
		oldUsage()
		fmt.Printf("\nArguments: [file1.json file2.json ... fileN.json]\n\n")
		fmt.Printf("Parses files generated by `go test -json f.json` and computes test set statistics.\n")
	}
	flag.Parse()

	args := flag.Args()

	switch statistic {
	case "":
		fmt.Printf("The `-statistic` flag is required.\n\n")
		flag.Usage()
	case "pkg-time":
		stats := newStatsFromFiles(args)
		pkgdurs := stats.packagesSortedByDurationDescending()
		for _, pkgdur := range pkgdurs {
			fmt.Printf("%s\t%v\n", pkgdur.id, pkgdur.duration)
		}
	case "test-time":
		stats := newStatsFromFiles(args)
		tests := stats.testsSortedByDurationDescending()
		for _, t := range tests {
			var status string
			if t.passed {
				status = "pass"
			} else {
				status = "fail"
			}
			fmt.Printf("%s\t%s\t%v\t%s\n", t.name, t.pkg, t.duration, status)
		}
	default:
		fmt.Printf("The `-statistic` flag is must be one of `pkg-time`, `test-time`.\n\n")
		flag.Usage()
	}
}

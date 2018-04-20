package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var slowTestRegexp = regexp.MustCompile(`^• \[SLOW TEST:(.*) seconds\]$`)
var fileNameRegexp = regexp.MustCompile(`.*(/test/extended/.*\.go.*)`)
var timeRegexp = regexp.MustCompile(`^([A-Z][a-z]{2}[ ]{1,2}[0-9]{1,2}[ ]{1,2}[0-9]{1,2}:[0-9]{2}:[0-9]{2}).*`)
var dockerTime = `^[0-9]{4}-[0-9]{2}-[0-9]{2}T([0-9]{2}:[0-9]{2}:[0-9]{2}).[0-9]*Z `
var dockerBuildStart = regexp.MustCompile(dockerTime + `Step 1/`)
var dockerBuildEnd = regexp.MustCompile(dockerTime + `Successfully built`)
var dockerPushStart = regexp.MustCompile(dockerTime + `Pushing image`)
var dockerPushEnd = regexp.MustCompile(dockerTime + `Push successful`)
var ignoreLines = []string{`INFO: Running AfterSuite actions on all node`}

type test struct {
	time       float64
	dockerInfo dockerInfo
	//blocks     []block
	lines []string
}

type blocks struct {
	offset int64
	Name   string  `json:"name"`
	Blocks []block `json:"block"`
}
type block struct {
	Lines     []string `json:"lines"`
	Start     int64    `json:"start"`
	End       int64    `json:"end"`
	BlockType string   `json:"blockType"`
}

type dockerBlock struct {
	Start     int64
	StartLine string
	End       int64
	EndLine   string
	BlockType string
}

type dockerInfo struct {
	blocks []dockerBlock
}

type stats struct {
	tests []test
}

type line struct {
	time    int64
	line    string
	hasTime bool
}

type window struct {
	timedWindow []line
	size        int
}

var out = flag.String("o", "out", "Output folder for bottlenecks")
var count = flag.Int("c", 5, "Show 'c' slowest tests")
var windowSize = flag.Int("w", 5, "Window size")
var threshold = flag.Int("t", 120, "Threshold in seconds to identify windows/bottleneck")
var file = flag.String("f", "file", "Log file to parse")

//var doubleDate = flag.Boolean("d", false, "May contain double date") //TODO:

func main() {
	flag.Parse()
	stats := parse(*file)
	printTop(stats)
	printStats(stats)
}

func (w *window) processLine(l string) bool {
	if m := timeRegexp.FindStringSubmatch(l); len(m) > 1 {
		t, _ := time.Parse(`Jan 2 15:04:05`, m[1])
		tl := line{t.Unix(), l, true}
		if len(w.timedWindow) < w.size {
			w.timedWindow = append(w.timedWindow, tl)
		} else {
			w.timedWindow = w.timedWindow[1:]
			w.timedWindow = append(w.timedWindow, tl)
		}
		return true
	}
	return false
}

func copyWin(w window) window {
	ntw := make([]line, len(w.timedWindow))
	copy(ntw, w.timedWindow)
	return window{ntw, len(w.timedWindow)}
}

func (w window) getTime() int64 {
	if len(w.timedWindow) == 0 {
		return 0
	}
	return w.timedWindow[len(w.timedWindow)-1].time - w.timedWindow[0].time
}

func (blcks *blocks) close(w window) {
	if len(blcks.Blocks) == 0 {
		blcks.Blocks = append(blcks.Blocks, block{})
	}
	b := &blcks.Blocks[len(blcks.Blocks)-1]
	b.End = w.timedWindow[len(w.timedWindow)-1].time - blcks.offset
	b.Lines = append(b.Lines, "...")
	b.Lines = append(b.Lines, w.timedWindow[len(w.timedWindow)-1].line)
	b.BlockType = "fast"
}

func (blcks *blocks) process(w window) {
	if len(blcks.Blocks) == 0 {
		blcks.Blocks = append(blcks.Blocks, block{})
		b := &blcks.Blocks[len(blcks.Blocks)-1]
		b.Lines = make([]string, 0)
	}
	b := &blcks.Blocks[len(blcks.Blocks)-1]
	if len(b.Lines) == 0 && len(w.timedWindow) > 0 {
		//init block with first timed line
		if len(blcks.Blocks) == 1 {
			blcks.offset = w.timedWindow[0].time
		}
		b.Start = w.timedWindow[0].time - blcks.offset
		b.Lines = append(b.Lines, w.timedWindow[0].line)
		return
	}
	if w.getTime() > int64(*threshold) {
		b.End = w.timedWindow[0].time - blcks.offset
		b.Lines = append(b.Lines, "...")
		b.Lines = append(b.Lines, w.timedWindow[0].line)
		b.BlockType = "fast"
		lines := make([]string, 0)
		for _, l := range w.timedWindow {
			lines = append(lines, l.line)
		}
		sb := block{
			lines,
			w.timedWindow[0].time - blcks.offset,
			w.timedWindow[len(w.timedWindow)-1].time - blcks.offset,
			"slow",
		}
		blcks.Blocks = append(blcks.Blocks, sb)
		fb := block{
			[]string{w.timedWindow[len(w.timedWindow)-1].line},
			w.timedWindow[len(w.timedWindow)-1].time - blcks.offset,
			0,
			"fast",
		}
		blcks.Blocks = append(blcks.Blocks, fb)
	}
}

func process(lines []string) ([]window, blocks) {
	w := make([]window, 0)
	b := blocks{0, "", make([]block, 0)}
	win := window{make([]line, 0), *windowSize}
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		win.processLine(l)
		b.process(win)
		if win.getTime() > int64(*threshold) {
			wc := copyWin(win)
			w = append(w, wc)
			wi := i + 1
			for j := 1; j <= *windowSize && wi < len(lines); wi++ {
				if win.processLine(lines[wi]) {
					j++
				}
			}
			i = wi
		}
	}
	b.close(win)
	sort.Slice(w, func(i, j int) bool { return w[i].getTime() > w[j].getTime() })
	return w, b
}

func getNames(i int, t test) (string, string) {
	fileName := "unknown"
	testName := "unknown"
	for _, l := range t.lines {
		m := fileNameRegexp.FindStringSubmatch(l)
		if len(m) > 1 {
			fileName = strings.Replace(m[1], `/`, `_`, -1)
			testName = m[1]
			break
		}
	}
	return *out + "/" + fmt.Sprintf("%04d", i) + "_" + fmt.Sprintf("%v", t.time) + fileName, testName
}

func writeResult(i int, t test) error {
	rf, _ := getNames(i, t)
	f, _ := os.Create(rf)
	defer f.Close()
	w := bufio.NewWriter(f)
	defer w.Flush()
	fmt.Fprintf(w, "time: %vs\n", t.time)
	for _, b := range t.dockerInfo.blocks {
		if b.End == -1 {
			b.End = b.Start
		}
		fmt.Fprintf(w, "docker %v: %vs\n  %v\n  %v\n",
			b.BlockType,
			b.End-b.Start,
			b.StartLine,
			b.EndLine)
	}
	windows, _ := process(t.lines)
	for i, b := range windows {
		fmt.Fprintf(w, "\nWindow %v - %vs\n", i, b.getTime())
		for _, l := range b.timedWindow {
			fmt.Fprintf(w, "%v\n", l.line)
		}
	}

	fmt.Fprintf(w, "\n\nEntire output:\n")
	for _, l := range t.lines {
		fmt.Fprintf(w, "%v\n", l)
	}
	return nil
}

func printTop(s stats) {
	if _, err := os.Stat("./" + *out); os.IsNotExist(err) {
		os.Mkdir("./"+*out, 0777)
	}

	sort.Slice(s.tests, func(i, j int) bool { return s.tests[i].time > s.tests[j].time })
	if *count < 1 {
		*count = len(s.tests)
	}
	for i := 0; i < *count && i < len(s.tests); i++ {
		writeResult(i+1, s.tests[i])
	}
}

func ignore(line string) bool {
	for _, l := range ignoreLines {
		if strings.Contains(line, l) {
			return true
		}
	}
	return false
}

func parse(f string) stats {
	file, err := os.Open(f)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	stats := stats{make([]test, 0)}
	scanner := bufio.NewScanner(file)
	buffer := make([]string, 0)
	dockerInfo := newDockerInfo()
	for scanner.Scan() {
		line := scanner.Text()
		if ignore(line) {
			continue
		}
		if strings.HasPrefix(line, "• [SLOW TEST:") {
			//end
			time, err := strconv.ParseFloat(slowTestRegexp.FindStringSubmatch(line)[1], 64)
			if err != nil {
				panic(err)
			}
			stats.tests = append(stats.tests, test{time, dockerInfo, buffer})
			buffer = make([]string, 0)
			dockerInfo = newDockerInfo()
		} else if strings.HasPrefix(line, "------------------------------") {
			//start
			buffer = make([]string, 0)
			dockerInfo = newDockerInfo()
		} else {
			//middle
			dockerInfo.parseDockerInfo(line)
		}
		buffer = append(buffer, line)
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}
	return stats
}

func newDockerInfo() dockerInfo {
	return dockerInfo{make([]dockerBlock, 0, 0)}
}

func getDockerTime(re *regexp.Regexp, line string) int64 {
	if m := re.FindStringSubmatch(line); len(m) > 1 {
		t, _ := time.Parse(`15:04:05`, m[1])
		return t.Unix()
	}
	return -1
}

func (di *dockerInfo) parseDockerInfo(line string) {
	if time := getDockerTime(dockerBuildStart, line); time != -1 {
		b := dockerBlock{time, line, -1, "", "build"}
		di.blocks = append(di.blocks, b)
		return
	}
	if time := getDockerTime(dockerPushStart, line); time != -1 {
		b := dockerBlock{time, line, -1, "", "push"}
		di.blocks = append(di.blocks, b)
		return
	}
	if time := getDockerTime(dockerBuildEnd, line); time != -1 {
		b := &(di.blocks[len(di.blocks)-1])
		b.End = time
		b.EndLine = line
		return
	}
	if time := getDockerTime(dockerPushEnd, line); time != -1 {
		b := &(di.blocks[len(di.blocks)-1])
		b.End = time
		b.EndLine = line
		return
	}
}

func printStats(s stats) {
	f, _ := os.Create(*out + "/stats.json")
	defer f.Close()
	w := bufio.NewWriter(f)
	defer w.Flush()
	if *count < 1 || *count > len(s.tests) {
		*count = len(s.tests)
	}
	tests := s.tests[0:*count]
	allBlocks := make([]blocks, 0)
	for i, t := range tests {
		_, n := getNames(i, t)
		_, blocks := process(t.lines)
		blocks.Name = n
		allBlocks = append(allBlocks, blocks)
	}
	if json, err := json.MarshalIndent(allBlocks, "", "  "); err == nil {
		w.Write(json)
	} else {
		fmt.Fprintf(w, "ERR: %v", err)
	}
}

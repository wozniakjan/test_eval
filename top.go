package main

import (
    "bufio"
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
var interestingRegexp = regexp.MustCompile(`^(STEP:)|(\[It\])`)

type test struct {
    time float64
    lines []string
}

type stats struct {
    tests []test
}

type line struct {
    time int64
    line string
    hasTime bool
}

type window struct {
    window []line
    timedWindow []line
    size int
}

var out = flag.String("o", "out", "Output folder for bottlenecks")
var count = flag.Int("c", 5, "Show 'c' slowest tests")
var windowSize = flag.Int("w", 5, "Window size")
var threshold = flag.Int("t", 120, "Threshold in seconds to identify windows/bottleneck")
var file = flag.String("f", "file", "Log file to parse")

func main() {
    flag.Parse()
    stats := parse(*file)
    printTop(stats)
}

func (w *window) processLine(l string) bool {
    if m := timeRegexp.FindStringSubmatch(l); len(m) > 1 {
        t, _ := time.Parse(`Jan 2 15:04:05`, m[1])
        tl := line{t.Unix(), l, true}
        if len(w.timedWindow) < w.size {
            w.timedWindow = append(w.timedWindow, tl)
            w.window = append(w.window, tl)
        } else {
            w.timedWindow = w.timedWindow[1:]
            w.window = w.window[1:]
            for ! w.window[0].hasTime {
                w.window = w.window[1:]
            }
            w.timedWindow = append(w.timedWindow, tl)
            w.window = append(w.window, tl)
        }
        return true
    } else {
        if m := interestingRegexp.FindStringSubmatch(l); len(m) > 1 {
            w.window = append(w.window, line{0, l, false})
        }
    }
    return false
}

func copyAndAppend(lines []string, ci int, w window) window {
    nw := make([]line, len(w.window))
    ntw := make([]line, len(w.timedWindow))
    copy(nw, w.window)
    copy(ntw, w.timedWindow)
    newWindow := window{nw, ntw, len(w.timedWindow)*2-1}
    pi := 1
    for i := ci+1; i < len(lines) && pi < len(w.timedWindow); i++ {
        if newWindow.processLine(lines[i]) {
            pi++
        }
    }
    return newWindow
}

func (w window) getWindowTime() int64 {
    return w.timedWindow[len(w.timedWindow)-1].time - w.timedWindow[0].time
}

func (w window) getTime() int64 {
    if len(w.timedWindow) == w.size {
        return w.getWindowTime()
    }
    return 0
}

func windows(lines []string) []window {
    b := make([]window,0)
    win := window{ make([]line, 0), make([]line, 0, *windowSize), *windowSize }
    lastI, ci := 0, 0
    for i, l := range lines {
        if win.processLine(l) {
            ci++
        }
        if win.getTime() > int64(*threshold) {
            if lastI+*windowSize <= ci {
                lastI = ci
                wc := copyAndAppend(lines, i, win)
                b = append(b, wc)
            }
        }
    }
    sort.Slice(b, func(i,j int) bool { return b[i].getWindowTime() > b[j].getWindowTime() })
    return b
}

func getName(i int, t test) string {
    fileName := "unknown"
    for _, l := range t.lines {
        m := fileNameRegexp.FindStringSubmatch(l)
        if len(m) > 1 {
            fileName = strings.Replace(m[1], `/`, `_`, -1)
            break
        }
    }
    return *out+"/"+fmt.Sprintf("%04d", i)+"_"+fmt.Sprintf("%v",t.time)+fileName;
}

func writeResult(i int, t test) error {
    f, _ := os.Create(getName(i, t))
    defer f.Close()
    w := bufio.NewWriter(f)
    defer w.Flush()
    fmt.Fprintf(w, "time: %v\n", t.time)
    windows := windows(t.lines)
    for i, b := range windows {
        fmt.Fprintf(w, "\nWindow %v - %vs\n", i, b.getWindowTime())
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
    if _, err := os.Stat("./"+*out); os.IsNotExist(err) {
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

func parse(f string) stats {
    file, err := os.Open(f)
    if err != nil {
        panic(err)
    }
    defer file.Close()

    stats := stats{make([]test,0)}
    scanner := bufio.NewScanner(file)
    buffer := make([]string, 0)
    for scanner.Scan() {
        line := scanner.Text()
        if strings.HasPrefix(line, "• [SLOW TEST:") {
            time, err := strconv.ParseFloat(slowTestRegexp.FindStringSubmatch(line)[1], 64)
            if err != nil {
                panic(err)
            }
            stats.tests = append(stats.tests, test{time, buffer})
        } else if strings.HasPrefix(line, "------------------------------") {
            buffer = make([]string, 0)
        }
        buffer = append(buffer, line)
    }

    if err := scanner.Err(); err != nil {
        panic(err)
    }
    return stats
}

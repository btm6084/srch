package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	homedir "github.com/mitchellh/go-homedir"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/btm6084/utilities/fileUtil"
	"github.com/btm6084/utilities/inarray"
)

var (
	before       int
	after        int
	insensitive  bool
	inverse      bool
	isTerminal   bool
	fileNameOnly bool
	followSyms   bool

	fileNameColor   = "\x1b[96m%s\x1b[0m"
	searchTermColor = "\x1b[30;42m$1\x1b[0m"
	lineNumColor    = "\x1b[93m%d%s\x1b[0m"

	fileNameNoColor   = "%s"
	searchTermNoColor = "$1"
	lineNumNoColor    = "%d%s"

	ignoreDirs []string

	outlock sync.Mutex
)

// Config contains the structure of the configuration file located at /home/$USER/.srchrc/config.json
type Config struct {
	IgnoreDirs []string `json:"ignore-dirs"`
}

func main() {
	// Set up configuration
	home, _ := homedir.Dir()
	config := getConfig(home + "/.srchrc/config.json")

	// Parse option flags
	i := flag.Bool("i", false, "Case insensitive search")
	v := flag.Bool("v", false, "Return lines that do not match the search term")
	l := flag.Bool("l", false, "Print filenames with matches")
	f := flag.Bool("follow", false, "Follow symlinks")
	a := flag.Int("A", 0, "Return this many lines after the matching line")
	b := flag.Int("B", 0, "Return this many lines before the matching line")
	ig := flag.String("ignore-dir", "", "Comma separated list of directories to ignore (eg. --ignore-dir=vendor,node_modules)")
	flag.Parse()

	before = *b
	after = *a
	insensitive = *i
	inverse = *v
	fileNameOnly = *l
	followSyms = *f
	ignoreDirs = strings.Split(*ig, ",")
	ignoreDirs = append(ignoreDirs, config.IgnoreDirs...)

	// When inverse, before and after don't make sense. Ignore them.
	if inverse {
		before = 0
		after = 0
	}

	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		isTerminal = true
	} else {
		isTerminal = false
	}

	search, path := getArgs()

	search = fmt.Sprintf("(%s)", search)
	if insensitive {
		search = fmt.Sprintf("(?i)%s", search)
	}

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		stdInSearch(search)
	} else {
		fileSystemSearch(search, path)
	}
}

// Get Args parses the input for search term and path
func getArgs() (string, string) {
	// Parse command line required parameters
	args := os.Args

	var nonFlags []string

	for k, v := range args {
		if k == 0 || (len(v) > 0 && string(v[0]) == `-`) {
			continue
		}

		nonFlags = append(nonFlags, v)
	}

	if len(nonFlags) < 1 {
		println("srch [flags] <search> [directory='./']", "Missing Search Term")
		os.Exit(1)
	}

	search := nonFlags[0]
	path := "."

	if len(nonFlags) > 1 {
		path = strings.TrimRight(nonFlags[1], "/")
	}

	return search, path
}

func stdInSearch(search string) {
	var lines []string
	var lastWrite time.Time
	writeInterval := time.Millisecond * 100

	scanner := bufio.NewReader(os.Stdin)

	for {
		line, err := scanner.ReadString('\n')
		if err == io.EOF {
			break
		}

		if err != nil {
			return
		}

		lines = append(lines, strings.Trim(line, "\n"))

		// Periodically flush the buffer.
		if time.Since(lastWrite) > writeInterval {
			lastWrite = time.Now()
			matches, _ := processLines(lines, search)
			lines = []string{}
			if len(matches) > 0 {
				println(strings.Join(matches, ""))
			}

		}
	}

	// Flush the buffer.
	if len(lines) > 0 {
		matches, _ := processLines(lines, search)
		if len(matches) > 0 {
			println(strings.Join(matches, ""))
		}
	}
}

// DirFilter returns true if the file should be kept, false if it should be discarded.
func DirFilter(path, dirName string) bool {
	if inarray.Strings(dirName, ignoreDirs) >= 0 {
		return false
	}

	return fileUtil.DefaultDirectoryFilter(path, dirName)
}

func fileSystemSearch(search, path string) {
	if !fileUtil.IsDir(path) {
		os.Exit(1)
	}

	// Extract file list
	files := fileUtil.DirToArray(path, followSyms, fileUtil.DefaultFileFilter, DirFilter)
	active := 0

	c := make(chan bool)

	// Perform the search
	for _, file := range files {
		go processFile(file, search, c)
		active++

		if active >= 10 {
			<-c
			active--
		}
	}

	// Wait for the last batch of concurrency to wrap up.
	for i := 0; i < active; i++ {
		<-c
	}
}

// processFile will search for any instances of the search string and log out anything found.
func processFile(fileName, search string, c chan bool) {
	file, err := os.Open(fileName)
	if err != nil {
		c <- false
		return
	}
	defer file.Close()

	matches, hasMatches := searchFile(file, search)

	var fileNameOut = fileNameNoColor
	if isTerminal {
		fileNameOut = fileNameColor
	}

	switch true {
	case hasMatches && inverse && fileNameOnly:
		c <- true
		return
	case !hasMatches && inverse && fileNameOnly:
		fileOut := fmt.Sprintf(fileNameOut, strings.TrimLeft(fileName, "./"))
		println(strings.TrimLeft(fileOut, "./"))

		c <- true
		return
	case hasMatches && !inverse && fileNameOnly:
		fileOut := fmt.Sprintf(fileNameOut, strings.TrimLeft(fileName, "./"))
		println(strings.TrimLeft(fileOut, "./"))

		c <- true
		return
	case len(matches) == 0:
		c <- true
		return
	default:
		fileOut := fmt.Sprintf(fileNameOut, strings.TrimLeft(fileName, "./"))
		println(strings.TrimLeft(fileOut, "./"), strings.Join(matches, ""))

		c <- true
		return
	}
}

func searchFile(file *os.File, search string) ([]string, bool) {
	var lines []string

	scanner := bufio.NewReader(file)

	for {
		line, err := scanner.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return processLines(lines, search)
			}
			return nil, false
		}
		lines = append(lines, strings.Trim(line, "\n"))
	}
}

func processLines(lines []string, search string) ([]string, bool) {
	var matches []string
	var matched []int
	var hasMatches bool

	sre := regexp.MustCompile(search)

	for lineNum, line := range lines {
		lineMatches := sre.Match([]byte(line))
		hasMatches = hasMatches || lineMatches

		if !inverse && lineMatches {
			matched = append(matched, lineNum)
		}

		if inverse && !lineMatches {
			matched = append(matched, lineNum)
		}
	}

	var lineNumOut = lineNumNoColor
	if isTerminal {
		lineNumOut = lineNumColor
	}

	var searchTermOut = searchTermNoColor
	if isTerminal {
		searchTermOut = searchTermColor
	}

	for _, k := range matched {
		start := k - before
		if start < 0 {
			start = 0
		}

		end := k + 1 + after
		if end > len(lines) {
			end = len(lines)
		}

		for i, l := range lines[start:end] {
			n := start + i
			lnOut := ""
			if isTerminal {
				lnOut = fmt.Sprintf(lineNumOut, n+1, ":")
			}

			switch true {
			case inverse:
				break
			case n < k && isTerminal:
				lnOut = fmt.Sprintf(lineNumOut, n+1, "-")
			case n == k && isTerminal:
				l = sre.ReplaceAllString(lines[k], searchTermOut)
				lnOut = fmt.Sprintf(lineNumOut, n+1, ":")
			case n > k && isTerminal:
				lnOut = fmt.Sprintf(lineNumOut, n+1, "+")
			}

			matches = append(matches, fmt.Sprintf("%s %s\n", lnOut, l))
		}

		if before > 0 || after > 0 {
			matches = append(matches, "--\n")
		}
	}

	return matches, hasMatches
}

func println(lines ...string) {
	outlock.Lock()
	fmt.Println(strings.Join(lines, "\n"))
	outlock.Unlock()
}

func getConfig(fileName string) Config {
	var c Config
	var tmp map[string][]string

	data, _ := ioutil.ReadFile(fileName)
	json.Unmarshal(data, &tmp)

	c.IgnoreDirs = tmp["ignore-dir"]
	return c
}

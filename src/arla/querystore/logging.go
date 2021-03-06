package querystore

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/mgutz/ansi"
)

type logLevel int

// Logging levels in
const (
	UNKNOWN logLevel = iota
	ALL              // 1: lower number, more logs
	DEBUG
	INFO
	LOG
	WARN
	ERROR // 6: higher number, less logs
)

func (ll logLevel) Fprintln(w io.Writer, args ...interface{}) (n int, err error) {
	defer fmt.Fprintf(w, ansi.Reset)
	switch ll {
	case UNKNOWN:
		fmt.Fprintf(w, ansi.LightBlack)
	case ALL:
		fmt.Fprintf(w, ansi.LightBlack)
	case DEBUG:
		fmt.Fprintf(w, ansi.LightBlack)
	case INFO:
		fmt.Fprintf(w, ansi.Blue)
	case LOG:
		fmt.Fprintf(w, ansi.White)
	case WARN:
		fmt.Fprintf(w, ansi.Yellow)
	case ERROR:
		fmt.Fprintf(w, ansi.Red)
	}
	return fmt.Fprintln(w, args...)
}

// LogFormatter parses log output, prettyifys it and interprets
// certains errors to give more useful output then forwards it on
// to the given writer
type LogFormatter struct {
	Level logLevel
	src   *string
	io.Writer
}

// NewLogFormatter creates a new LogFormatter
func NewLogFormatter(w io.Writer) *LogFormatter {
	pr, pw := io.Pipe()
	log := &LogFormatter{Writer: pw}
	go func() {
		scanner := bufio.NewScanner(pr)
		var level logLevel
		for scanner.Scan() {
			var line string
			s := strings.Replace(scanner.Text(), "NOTICE:", "", 1)
			ss := strings.SplitN(s, ":", 2)
			if len(ss) < 2 {
				line = s
			} else {
				line = ss[1]
				switch strings.TrimSpace(ss[0]) {
				case "CONTEXT":
					level = ALL
				case "DEBUG":
					level = DEBUG
				case "INFO":
					level = INFO
				case "LOG":
					// we want to re-interupt some lines as debug
					if strings.Contains(line, "statement:") == true {
						level = DEBUG
					} else if strings.Contains(line, "execute <unnamed>:") == true {
						level = DEBUG
					} else {
						level = LOG
					}
				case "WARN", "WARNING":
					level = WARN
				case "ERROR":
					level = ERROR
				case "DETAIL":
					if strings.Contains(line, "plv8_init() LINE") {
						level = ERROR
					} else {
						level = DEBUG
					}
				case "FATAL":
					level = ERROR
				default:
					line = s
				}
			}
			if level == ERROR && log.src != nil {
				// try to extract better info about the error
				var plv8err = regexp.MustCompile(`LINE\s+(\d+):`)
				if plv8err.MatchString(line) {
					if lineNo, err := strconv.Atoi(plv8err.FindStringSubmatch(line)[1]); err == nil {
						// add offset to the plv8_init function
						lineNo += 4
						// if lineNo < postgresInitUserOffset {
						// 	continue
						// }
						// build context
						amt := 8
						var buf bytes.Buffer
						src := strings.Split(*log.src, "\n")
						start := lineNo - amt
						if start < 0 {
							start = 0
						}
						end := lineNo + amt
						if end > len(src)-1 {
							end = len(src) - 1
						}
						// for i := start; i < end; i++ {
						// 	xtra := ""
						// 	col := ansi.White
						// 	if i == lineNo {
						// 		col = ansi.LightWhite
						// 		xtra = fmt.Sprint(ansi.Reset, ansi.Red, "<--- something wrong here", ansi.Reset)
						// 	}
						// 	fmt.Fprintln(&buf, ansi.Reset, col, src[i], ansi.Reset, xtra)
						// }
						line = buf.String()
					}
				}
			}
			if level >= log.Level {
				level.Fprintln(w, line)
			}
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "log formatter said: ", err)
		}
	}()
	return log
}

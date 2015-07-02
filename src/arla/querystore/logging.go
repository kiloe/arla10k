package querystore

import (
	"bufio"
	"fmt"
	"io"
	"os"
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
	// fmt.Fprint(w, ll, "--> ")
	return fmt.Fprintln(w, args...)
}

// LogFormatter parses log output, prettyifys it and interprets
// certains errors to give more useful output then forwards it on
// to the given writer
type LogFormatter struct {
	Level logLevel
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
			s := strings.TrimSpace(scanner.Text())
			ss := strings.SplitN(s, ":", 2)
			if len(ss) < 2 {
				line = s
			} else {
				line = strings.TrimSpace(ss[1])
				switch ss[0] {
				case "DEBUG":
					level = DEBUG
				case "INFO":
					level = INFO
				case "LOG":
					// we want to re-interupt some lines as debug
					if strings.HasPrefix(line, "statement:") == true {
						level = DEBUG
					} else if strings.HasPrefix(line, "execute <unnamed>:") == true {
						level = DEBUG
					} else {
						level = LOG
					}
				case "WARN":
					level = WARN
				case "ERROR":
				case "FATAL":
					level = ERROR
				}
				if level == UNKNOWN {
					line = s
				}
			}
			if log.Level <= level {
				level.Fprintln(w, line)
			}
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "log formatter said: ", err)
		}
	}()
	return log
}

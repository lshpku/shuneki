package sla

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"
)

type LogLevel uint32

const (
	DEBUG LogLevel = 100
	INFO  LogLevel = 200
	WARN  LogLevel = 300
	ERROR LogLevel = 400
	FATAL LogLevel = 500
)

var (
	flagLogLevel    = flag.String("sla.log.level", "info", "may be from \"debug\" to \"fatal\", or integer in [0, 999]")
	flagLogDir      = flag.String("sla.log.dir", "", "log file dir. a blank value means logging to stderr only")
	flagLogRollLine = flag.Int("sla.log.roll.line", 65536, "max number of lines per log file. a negative value means no limit")
	flagLogRollResv = flag.Int("sla.log.roll.resv", 16, "max number of files to reserve. a negative value means no limit")

	logLevel          = INFO
	logTimeFormat     = "2006-01-02 15:04:05"
	logFileTimeFormat = "20060102_150405"
	logFile           *os.File
	logLine           int
	rollMutex         sync.Mutex
)

// Init creates log file and parses flags
func Init() {
	// create log file
	if len(*flagLogDir) != 0 {
		if err := rollLog(); err != nil {
			fmt.Fprintf(os.Stderr, "cannot create log file: %s\n", err.Error())
			os.Exit(1)
		}
	}

	// parse flags
	switch *flagLogLevel {
	case "debug":
		logLevel = DEBUG
	case "info":
		logLevel = INFO
	case "warn":
		logLevel = WARN
	case "error":
		logLevel = ERROR
	case "fatal":
		logLevel = FATAL
	default:
		level, err := strconv.Atoi(*flagLogLevel)
		if err != nil {
			commonLog("fata", "cannot parse sla.log.level: %s", *flagLogLevel)
			os.Exit(1)
		}
		if level < 0 || level > 999 {
			commonLog("fata", "sla.log.level must be in [0, 999], now: %d", level)
			os.Exit(1)
		}
		logLevel = LogLevel(level)
	}

	if *flagLogRollResv < 0 {
		Fatal("sla.log.roll.resv should be ")
	}
}

func Debug(format string, a ...interface{}) {
	if logLevel <= DEBUG {
		commonLog("debg", format, a...)
	}
}

func Info(format string, a ...interface{}) {
	if logLevel <= INFO {
		commonLog("info", format, a...)
	}
}

func Warn(format string, a ...interface{}) {
	if logLevel <= WARN {
		commonLog("warn", format, a...)
	}
}

func Error(format string, a ...interface{}) {
	if logLevel <= ERROR {
		commonLog("erro", format, a...)
	}
}

func Fatal(format string, a ...interface{}) {
	if logLevel <= FATAL {
		commonLog("fata", format, a...)
	}
	os.Exit(1)
}

func commonLog(level, format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	t := time.Now().Format(logTimeFormat)
	line := fmt.Sprintf("%s [%s] %s\n", t, level, msg)

	if len(*flagLogDir) != 0 {
		rollMutex.Lock()
		if logLine >= *flagLogRollLine {
			rollLog()
			logLine = 0
		}
		logLine++
		rollMutex.Unlock()
		fmt.Fprint(logFile, line)
	}

	fmt.Print(line)
}

type strArr []string

func (s strArr) Len() int {
	return len(s)
}

func (s strArr) Less(i, j int) bool {
	return s[i] < s[j]
}

func (s strArr) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// rollLog delete obsolete log file(s) and create a new log file
func rollLog() error {
	logDir, err := filepath.Abs(*flagLogDir)
	if nil != err {
		return err
	}

	fis, err := ioutil.ReadDir(logDir)
	if nil != err {
		return err
	}

	fns := make(strArr, 0, len(fis))

	for _, fi := range fis {
		if fi.IsDir() {
			continue
		}

		// match file name
		fn := fi.Name()
		if len(fn) != 19 {
			continue
		}
		if fn[:4] != "log_" {
			continue
		}
		_, err := time.Parse(logFileTimeFormat, fn[4:])
		if err != nil {
			continue
		}
		fns = append(fns, fn)
	}

	if len(fns) < *flagLogRollResv {
		return nil
	}

	// sort by timestamp in file name
	sort.Sort(fns)

	for i := 0; i+*flagLogRollResv <= len(fns); i++ {
		os.Remove(filepath.Join(*flagLogDir, fns[i]))
	}

	// create new log file
	fn := fmt.Sprintf("log_%s", time.Now().Format(logFileTimeFormat))
	newLogFile, err := os.Create(filepath.Join(*flagLogDir, fn))
	if err != nil {
		return err
	}
	logFile = newLogFile

	return nil
}

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
	flagLogRollLine = flag.Int("sla.log.roll.line", 0, "max number of lines per log file. a value < 1 means no limit")
	flagLogRollResv = flag.Int("sla.log.roll.resv", 0, "max number of files to reserve. a value < 1 means no limit")

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
			// print to stderr only
			Fatal("cannot create log file: %s\n", err.Error())
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
			Fatal("cannot parse sla.log.level: %s", *flagLogLevel)
		}
		if level < 0 || level > 999 {
			Fatal("sla.log.level must be in [0, 999], now: %d", level)
		}
		logLevel = LogLevel(level)
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

	if logFile != nil {
		if *flagLogRollLine > 0 {
			// TODO: may cause a long wait
			rollMutex.Lock()
			logLine++

			// need rolling
			if logLine > *flagLogRollLine {
				err := rollLog()
				if err != nil {
					// if rollLog fails, use stderr from now on
					logFile = nil
					Warn("cannot roll log. use stderr from now on: %s", err.Error())
				} else {
					logLine = 1
				}
			}
			rollMutex.Unlock()
		}
		fmt.Fprint(logFile, line)
	}

	fmt.Fprintf(os.Stderr, line)
}

type timeArr []time.Time

func (a timeArr) Len() int {
	return len(a)
}

func (a timeArr) Less(i, j int) bool {
	return a[i].Before(a[j])
}

func (a timeArr) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

// rollLog deletes obsolete log files and replaces logFile with a new one.
// It will not print any logs, but return an error when rolling cannot be
// done
func rollLog() error {
	if *flagLogRollResv > 0 {
		fileInfos, err := ioutil.ReadDir(*flagLogDir)
		if nil != err {
			return err
		}
		fileTimes := make(timeArr, 0, len(fileInfos))

		// find all files that match log file name
		for _, fi := range fileInfos {
			if fi.IsDir() {
				continue
			}
			fn := fi.Name()
			if len(fn) != 19 || fn[:4] != "log_" {
				continue
			}
			t, err := time.Parse(logFileTimeFormat, fn[4:])
			if err != nil {
				continue
			}
			fileTimes = append(fileTimes, t)
		}

		if len(fileTimes) >= *flagLogRollResv {
			// sort by timestamp in file name
			sort.Sort(fileTimes)

			// delete anyway. Don't care if some files have been deleted
			for i := 0; len(fileTimes)-i >= *flagLogRollResv; i++ {
				fn := "log_" + fileTimes[i].Format(logFileTimeFormat)
				os.Remove(filepath.Join(*flagLogDir, fn))
			}
		}
	}

	// create new log file
	fn := "log_" + time.Now().Format(logFileTimeFormat)
	newLogFile, err := os.Create(filepath.Join(*flagLogDir, fn))
	if err != nil {
		return err
	}
	if logFile != nil {
		logFile.Close()
	}
	logFile = newLogFile

	return nil
}

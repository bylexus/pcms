package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flosch/pongo2/v4"
)

type Level int

const (
	DEBUG   = 0
	INFO    = 1
	WARNING = 2
	ERROR   = 3
	FATAL   = 4
)

func StrToLevel(level string) Level {
	switch strings.ToLower(level) {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warning":
		return WARNING
	case "error":
		return ERROR
	case "fatal":
		return FATAL
	default:
		return DEBUG
	}
}

func LevelToStr(level Level) string {
	switch level {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARNING:
		return "WARNING"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

type Logger struct {
	Filepath string
	level    Level
	file     *os.File
	format   string
	template *pongo2.Template
}

func NewLogger(filename string, level Level, format string) *Logger {
	var f *os.File
	var fpath = ""
	var err error
	if len(format) == 0 {
		format = defaultFormat()
	}

	if strings.ToLower(filename) == "stdout" {
		f = os.Stdout
		fpath = "STDOUT"
	} else if strings.ToLower(filename) == "stderr" {
		f = os.Stderr
		fpath = "STDERR"
	} else {
		fpath, _ = filepath.Abs(filename)
		fdir := filepath.Dir(fpath)
		err = os.MkdirAll(fdir, os.ModeDir)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Using logfile: %s: %s\n", filename, fpath)
		f, err = os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			panic(err)
		}
	}
	tpl, err := pongo2.FromString(format)
	if err != nil {
		panic(err)
	}

	l := Logger{
		Filepath: fpath,
		level:    level,
		file:     f,
		format:   format,
		template: tpl,
	}
	return &l
}

// supported formats (pongo2 template syntax):
// {{time}} outputs the current time - in RFC 3339 format
// {{level}} outputs the requested logging level
// {{message}} outputs the message logged
func defaultFormat() string {
	return "{{time}} {{level}} {{message}}"
}

func (l *Logger) Close() {
	if l.file != nil {
		l.file.Close()
	}
}

func (l *Logger) Log(msg string, level Level, msgParams ...interface{}) {
	if level < l.level {
		return
	}

	now := time.Now().Format(time.RFC3339)

	out, _ := l.template.Execute(pongo2.Context{
		"time":    now,
		"level":   LevelToStr(level),
		"message": fmt.Sprintf(msg, msgParams...),
	})
	l.file.WriteString(out + "\n")
}

func (l *Logger) Debug(msg string, msgParams ...interface{}) {
	l.Log(msg, DEBUG, msgParams...)
}

func (l *Logger) Info(msg string, msgParams ...interface{}) {
	l.Log(msg, INFO, msgParams...)
}

func (l *Logger) Warning(msg string, msgParams ...interface{}) {
	l.Log(msg, WARNING, msgParams...)
}

func (l *Logger) Error(msg string, msgParams ...interface{}) {
	l.Log(msg, ERROR, msgParams...)
}

func (l *Logger) Fatal(msg string, msgParams ...interface{}) {
	l.Log(msg, FATAL, msgParams...)
	panic(fmt.Sprintf(msg, msgParams...))
}

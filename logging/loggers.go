package logging

import (
	"log"
	"os"
)

type Loggers struct {
	logInfo    *log.Logger
	logWarning *log.Logger
	logError   *log.Logger
}

func NewLoggers() *Loggers {
	logInfo := log.New(os.Stdout, "[INFO] ", log.LstdFlags)
	logWarning := log.New(os.Stdout, "[WARN] ", log.LstdFlags)
	logError := log.New(os.Stdout, "[ERROR] ", log.LstdFlags)

	return &Loggers{
		logInfo:    logInfo,
		logWarning: logWarning,
		logError:   logError,
	}
}

func (l *Loggers) Info(v ...any) {
	l.logInfo.Println(v...)
}

func (l *Loggers) Warning(v ...any) {
	l.logWarning.Println(v...)
}

func (l *Loggers) Error(v ...any) {
	l.logError.Println(v...)
}

func (l *Loggers) Infof(format string, v ...any) {
	l.logInfo.Printf(format, v...)
}

func (l *Loggers) Warningf(format string, v ...any) {
	l.logWarning.Printf(format, v...)
}

func (l *Loggers) Errorf(format string, v ...any) {
	l.logError.Printf(format, v...)
}

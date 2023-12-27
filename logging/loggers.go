package logging

import (
	"log"
	"os"
)

var (
	logInfo    *log.Logger
	logWarning *log.Logger
	logError   *log.Logger
)

func init() {
	logInfo = log.New(os.Stdout, "[INFO] ", log.LstdFlags)
	logWarning = log.New(os.Stdout, "[WARN] ", log.LstdFlags)
	logError = log.New(os.Stdout, "[ERROR] ", log.LstdFlags)
}

func Info(v ...any) {
	logInfo.Println(v...)
}

func Warning(v ...any) {
	logWarning.Println(v...)
}

func Error(v ...any) {
	logError.Println(v...)
}

func Infof(format string, v ...any) {
	logInfo.Printf(format, v...)
}

func Warningf(format string, v ...any) {
	logWarning.Printf(format, v...)
}

func Errorf(format string, v ...any) {
	logError.Printf(format, v...)
}

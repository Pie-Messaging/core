package pie

import (
	"log"
	"os"
)

var (
	Logger     = &log.Logger{}
	loggerFile *os.File
)

func Log(noLog []bool, v ...any) {
	if len(noLog) == 0 || !noLog[0] {
		Logger.Println(v...)
	}
}

func SetLogOutput(output string) {
	if output == "" {
		Logger.SetOutput(os.Stdout)
		return
	}
	var err error
	loggerFile, err = os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o666)
	if err != nil {
		log.Println("Failed to open log file:", err)
	}
	Logger.SetOutput(loggerFile)
}

func CloseLogFile() {
	err := loggerFile.Close()
	if err != nil {
		log.Println("Failed to close log file:", err)
	}
}

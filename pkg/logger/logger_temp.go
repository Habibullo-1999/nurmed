package logger

import (
	"log"
)

// New constructs a new logger.
//func NewLogger2() ILogger {
//
//	const file = "./app.log"
//
//	log.SetOutput(&lumberjack.Logger{
//		Filename:  file,
//		Compress:  true, // disabled by default
//		LocalTime: true,
//	})
//
//	return &logger2{}
//
//}

type logger2 struct {
}

func (l *logger2) Debug(format string, v ...interface{}) {
	log.Printf("DEBUG:"+format, v...)
}

func (l *logger2) Info(format string, v ...interface{}) {
	log.Printf("INFO:"+format, v...)
}

func (l *logger2) Warning(format string, v ...interface{}) {
	log.Printf("WARNING:"+format, v...)
}

func (l *logger2) Error(format string, v ...interface{}) {
	log.Printf("ERROR:"+format, v...)
}

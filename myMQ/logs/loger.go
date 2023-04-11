// Package lg provides leveled logging
package logs

import (
	"fmt"
	"log"
	"os"
)

type Level int

// 枚举日志等级
const (
	DEBUG Level = iota
	INFO
	WARNING
	ERROR
	FATAL
)

// flags
const (
	Ldate         = log.Ldate         //eg:2009/01/23
	Ltime         = log.Ltime         //eg:01:23:23
	Lmicroseconds = log.Lmicroseconds //01:23:23.123123
	Llongfile     = log.Llongfile     //完整的文件名和行号
	Lshortfile    = log.Lshortfile    //最后一个文件名和行号
	LstdFlags     = log.LstdFlags     //标准（默认）
)

// 配置（目前较为简单）
var (
	LogLevel  Level       //日志等级
	outfil    os.File     //输出文件
	logPrefix string      //日志前缀
	logger    *log.Logger //向终端的输出
	loggerf   *log.Logger //向文件输出（可设置）
)

// 用来设置前缀
var levelFlags = []string{"DEBUG", "INFO", "WARNING", "ERROR", "FATAL"}

// 初始化默认配置
func init() {
	LogLevel = DEBUG
	logPrefix = ""
	logger = log.New(os.Stdout, "[default]", LstdFlags)
}

// Println ..
func Println(l *log.Logger, v ...interface{}) {
	if l != nil {
		l.Output(3, fmt.Sprintln(v...))
	}
}
func Printf(l *log.Logger, format string, v ...interface{}) {
	if l != nil {
		l.Output(3, fmt.Sprintf(format, v...))
	}
}

// Fatalln is equivalent to l.Println() followed by a call to os.Exit(1).
func Fatalln(l *log.Logger, v ...interface{}) {
	if l != nil {
		l.Output(3, fmt.Sprintln(v...))
		os.Exit(1)
	}
}

func Fatalf(l *log.Logger, format string, v ...interface{}) {
	if l != nil {
		l.Output(3, fmt.Sprintf(format, v...))
		os.Exit(1)
	}
}

// Debug ...
func Debug(format string, v ...interface{}) {
	setPrefix(DEBUG)
	if DEBUG >= LogLevel {
		Printf(logger, format, v...)
		Printf(loggerf, format, v...)
	}

}

// Info ...
func Info(format string, v ...interface{}) {
	setPrefix(INFO)
	if INFO >= LogLevel {
		Printf(logger, format, v...)
		Printf(loggerf, format, v...)
	}
}

// Warn ...
func Warn(format string, v ...interface{}) {
	setPrefix(WARNING)
	if WARNING >= LogLevel {
		Printf(logger, format, v...)
		Printf(loggerf, format, v...)
	}
}

// Error Warn
func Error(format string, v ...interface{}) {
	setPrefix(ERROR)
	if ERROR >= LogLevel {
		Printf(logger, format, v...)
		Printf(loggerf, format, v...)
	}
}

// Fatal ...
func Fatal(v ...interface{}) {
	setPrefix(FATAL)
	if FATAL >= LogLevel {
		Fatalln(logger, v...)
		Println(loggerf, v...)
	}

}
func setPrefix(level Level) {
	logPrefix = fmt.Sprintf("[%s] ", levelFlags[level])
	logger.SetPrefix(logPrefix)
	if loggerf != nil {
		loggerf.SetPrefix(logPrefix)
	}
}

// Config ..
func Config(level Level, lfile *os.File) {
	LogLevel = level
	if lfile != nil {
		loggerf = log.New(lfile, "[default] ", log.LstdFlags)
		loggerf.SetFlags(log.Ldate | log.Llongfile)
	}
}

func SetFlags(flag int) {
	logger.SetFlags(flag)
	if loggerf != nil {
		loggerf.SetFlags(flag)
	}
}

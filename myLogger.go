package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
	"time"
)

// LogLevel 日志级别
type LogLevel uint16

var (
	// MaxSize 日志通道的大小
	MaxSize int = 10000
)

//
const (
	UNKNOWN LogLevel = iota
	DEBUG
	TRACE
	Info
	WARNING
	ERROR
	FATAL
)

func parseloglevel(s string) (LogLevel, error) {
	s = strings.ToLower(s) // 强转
	switch s {
	case "debug":
		return DEBUG, nil
	case "trace":
		return TRACE, nil
	case "info":
		return Info, nil
	case "warning":
		return WARNING, nil
	case "error":
		return ERROR, nil
	case "fatal":
		return FATAL, nil
	default:
		err := errors.New("无效的日志级别")
		return UNKNOWN, err
	}
}

func getlogString(lv LogLevel) string {
	switch lv {
	case DEBUG:
		return "DEBUG"
	case TRACE:
		return "TRACE"
	case Info:
		return "Info"
	case WARNING:
		return "WARNING"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	}
	return "DEBUG"
}

func getInfo(skip int) (funcName, fileName string, lineNO int) {
	pc, file, lineNO, ok := runtime.Caller(skip)
	if ok == false {
		fmt.Printf("runetime.Caller() failed\n")
		return
	}
	funcName = runtime.FuncForPC(pc).Name()
	fileName = path.Base(file)
	fileName = strings.Split(file, ".")[1]
	return
}

// ConsoleLogger 日志结构体
type ConsoleLogger struct {
	Level LogLevel
}

//判断是否需要记录该日志
func (f *Filelogger) enable(levellog LogLevel) bool {
	return levellog >= f.level
}

// 判断文件时候需求切割
func (f *Filelogger) checkSize(file *os.File) bool {
	fileInfo, err := file.Stat() //获取文件的大小
	if err != nil {
		return false
	}
	//比较文件大小，如果这个文件大于原始文件，则返回true
	return fileInfo.Size() >= f.maxFileSize
}

func (f *Filelogger) writeLogBackground() {
	for {
		if f.checkSize(f.fileObj) {
			newFile, err := f.spitfile(f.fileObj)
			if err != nil {
				return
			}
			f.fileObj = newFile
		}
		select {
		case logTmp := <-f.logChan:
			logInfo := fmt.Sprintf("[%s] [%s] [%s: %s: %d] %s\n", logTmp.timeStamp, getlogString(logTmp.Level), logTmp.fileName, logTmp.funcName, logTmp.line, logTmp.msg)
			fmt.Fprintf(f.fileObj, logInfo)
		default:
			// 取不到日志休息500毫秒
			time.Sleep(time.Millisecond * 500)
		}
	}
}

// 写日志
func (f *Filelogger) log(lv LogLevel, format string, a ...interface{}) {
	if f.enable(lv) {
		msg := fmt.Sprintf(format, a...) // 格式化赋值给msg
		now := time.Now()
		funcName, fileName, lineNo := getInfo(3)
		// 发送日志到通道中
		logTmp := &logMsg{
			Level:     lv,
			msg:       msg,
			funcName:  funcName,
			fileName:  fileName,
			timeStamp: now.Format("2006-01-02 15:04:05"),
			line:      lineNo,
		}
		select {
		case f.logChan <- logTmp:
		default:
			// 通道满了，日志丢掉
		}
	}
}

//切割文件
func (f Filelogger) spitfile(file *os.File) (*os.File, error) {
	// 关闭当前的日志文件
	file.Close()
	// 创建一个新的日志文件
	newLogName := time.Now().Format("20060102150405") + ".log"
	fileObj, err := os.OpenFile(newLogName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf(" open new log file failed err:%v\n", err)
		return nil, err
	}
	// 将打开的新日志文件对象赋值给 f.fileobj
	return fileObj, nil
}

// Logger ...
type Logger interface {
	Debug(format string, a ...interface{})
	Info(format string, a ...interface{})
	Warning(format string, a ...interface{})
	Error(format string, a ...interface{})
	Fatal(format string, a ...interface{})
}

// Debug ...
func (f *Filelogger) Debug(format string, a ...interface{}) {
	f.log(DEBUG, format, a...)
}

// Info ...
func (f *Filelogger) Info(format string, a ...interface{}) {

	f.log(Info, format, a...)

}

// Warning ...
func (f *Filelogger) Warning(format string, a ...interface{}) {
	f.log(WARNING, format, a...)

}

// Error ...
func (f *Filelogger) Error(format string, a ...interface{}) {
	f.log(ERROR, format, a...)

}

// Fatal ...
func (f *Filelogger) Fatal(format string, a ...interface{}) {
	f.log(FATAL, format, a...)

}

// Newlog 构造函数
func Newlog(levelStr string) ConsoleLogger {
	level, err := parseloglevel(levelStr)
	if err != nil {
		panic(err)
	}
	return ConsoleLogger{
		Level: level,
	}

}

// Filelogger 往文件里面写日志
type Filelogger struct {
	level       LogLevel
	filepath    string // 日志保存的路径
	filename    string //日志保存的名称
	fileObj     *os.File
	maxFileSize int64
	logChan     chan *logMsg
}

type logMsg struct {
	Level     LogLevel
	msg       string
	funcName  string
	fileName  string
	timeStamp string
	line      int
}

// NewFileLogger 初始化日志
func NewFileLogger(levelStr, fp string, maxSize int64) *Filelogger {
	LogLevel, err := parseloglevel(levelStr)
	if err != nil {
		panic(err)
	}
	logName := time.Now().Format("20060102030405") + ".log"
	fl := &Filelogger{
		level:       LogLevel,
		filepath:    fp,
		filename:    logName,
		maxFileSize: maxSize,
		logChan:     make(chan *logMsg, MaxSize),
	}
	err = fl.initfile()
	if err != nil {
		panic(err)
	}
	return fl //按照文件路径和文件名打开
}

// 根据指定的日志文件路径和文件名打开
func (f *Filelogger) initfile() error {
	fullFileName := path.Join(f.filepath, f.filename)
	fileObj, err := os.OpenFile(fullFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("open log file failed ,err:%v", err)
		return err
	}
	f.fileObj = fileObj

	// 开启后台的goroutine去写日志
	go f.writeLogBackground()

	return nil
}

// Close 关闭文件
func (f *Filelogger) Close() {
	f.fileObj.Close()
}

var log Logger

func main() {
	// 10*1024 10k测试文件分割
	log = NewFileLogger("info", "./", 10*1024)
	for {
		log.Debug("这是Debug日志")
		log.Info("这是Info日志")
		log.Warning("这是Warning日志")
		id := 1000
		name := "嘿 嘿"
		log.Error("这是Error日志,%d,%s", id, name)
		log.Fatal("这是Fatal日志")
		time.Sleep(time.Second)
	}
}

package common

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type LoggerInterface interface {
	Verbose(s ...interface{})
	Debug(s ...interface{})
	Info(s ...interface{})
	Warn(s ...interface{})
	Error(s ...interface{})
	Fatal(s ...interface{})
	FatalGo(s ...interface{})
}

type LoggerEmpty struct{}

func (*LoggerEmpty) Verbose(s ...interface{}) {}
func (*LoggerEmpty) Debug(s ...interface{})   {}
func (*LoggerEmpty) Info(s ...interface{})    {}
func (*LoggerEmpty) Warn(s ...interface{})    {}
func (*LoggerEmpty) Error(s ...interface{})   {}
func (*LoggerEmpty) Fatal(s ...interface{})   {}
func (*LoggerEmpty) FatalGo(s ...interface{}) {}

type logLevels int

const (
	_ int = iota + 90 // fgHiBlack
	fgHiRed
	fgHiGreen
	_ // fgHiYellow
	fgHiBlue
	fgHiMagenta
	fgHiCyan
	_ // fgHiWhite

	// LevelDebug самый подробный лог
	LevelDebug logLevels = iota
	// LevelVerbose подробный лог, но без дебагг-инфо
	LevelVerbose
	// LevelInfo только ошибки, предупреждения и информация
	LevelInfo
	// LevelWarn только ошибки и предупреждения
	LevelWarn
	// LevelError только ошибки
	LevelError
	// LevelFatal ошибка приводит к завершению приложения
	LevelFatal
)

type logType struct {
	prefix string
	color  int
}

var (
	logTypes map[logLevels]logType
	// Глобальный объект-логгинга
	Log *Logger
	// IsDev   bool
	// IsProd  bool
	// IsMacOs bool
)

func init() {

	logTypes = map[logLevels]logType{
		LevelFatal:   logType{"ftl", fgHiRed},
		LevelError:   logType{"err", fgHiRed},
		LevelWarn:    logType{"wrn", fgHiMagenta},
		LevelInfo:    logType{"inf", fgHiGreen},
		LevelVerbose: logType{"vrb", fgHiCyan},
		LevelDebug:   logType{"dbg", fgHiBlue},
	}

	Log = NewLogger(os.Stdout, LevelDebug)

	// IsMacOs = runtime.GOOS == "darwin"
	// IsDev = IsMacOs
	// IsProd = !IsDev
	Log.Info("starting...")
}

// Logger тип
type Logger struct {
	out            []io.Writer
	level          logLevels
	mutex          sync.Mutex
	buf            []byte
	useColors      bool
	callStackAdder int
	noFilename     bool
	customFilename string
}

// NewLogger Создает новый логгер
//	* out		- io.Writer
//	* level	- уровень логгинга
func NewLogger(out io.Writer, level logLevels) *Logger {

	return &Logger{out: []io.Writer{out}, level: level, useColors: true}
}

func itoa(buf *[]byte, i int, width int) {
	var b [20]byte
	bp := len(b) - 1
	for i >= 10 || width > 1 {
		width--
		q := i / 10
		b[bp] = byte('0' + i - q*10)
		bp--
		i = q
	}
	// i < 10
	b[bp] = byte('0' + i)
	*buf = append(*buf, b[bp:]...)
}

func (l *Logger) writeToOut(level logLevels, message string) {

	now := time.Now()
	logType := logTypes[level]

	_, file, line, ok := runtime.Caller(3 + l.callStackAdder)
	// pc, file, line, ok := runtime.Caller(3 + l.callStackAdder)
	// fn := runtime.FuncForPC(pc)

	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.buf = l.buf[:0]
	if l.useColors {
		l.buf = append(l.buf, "\x1b[1;"...)
		itoa(&l.buf, logType.color, 2)
		l.buf = append(l.buf, 'm')
	}

	l.buf = append(l.buf, "    "...)
	l.buf = append(l.buf, logType.prefix...)

	l.buf = append(l.buf, ' ')

	year, month, day := now.Date()
	itoa(&l.buf, year, 4)
	l.buf = append(l.buf, '-')
	itoa(&l.buf, int(month), 2)
	l.buf = append(l.buf, '-')
	itoa(&l.buf, day, 2)
	l.buf = append(l.buf, ' ')

	hour, min, sec := now.Clock()
	itoa(&l.buf, hour, 2)
	l.buf = append(l.buf, ':')
	itoa(&l.buf, min, 2)
	l.buf = append(l.buf, ':')
	itoa(&l.buf, sec, 2)
	l.buf = append(l.buf, '.')
	itoa(&l.buf, now.Nanosecond()/1e3, 6)

	if !l.noFilename {
		if l.customFilename != "" {
			l.buf = append(l.buf, " ["...)
			l.buf = append(l.buf, l.customFilename...)
			l.customFilename = ""
			l.buf = append(l.buf, "]"...)
		} else if ok {
			l.buf = append(l.buf, " ["...)
			l.buf = append(l.buf, filepath.Base(file)...)
			l.buf = append(l.buf, ':')
			itoa(&l.buf, line, -1)
			// if fn != nil {
			// 	l.buf = append(l.buf, ':')
			// 	l.buf = append(l.buf, fn.Name()...)
			// }
			l.buf = append(l.buf, "]"...)
		}
	}

	if l.useColors {
		l.buf = append(l.buf, "\x1b[0m"...)
	}
	l.buf = append(l.buf, ' ')
	l.buf = append(l.buf, message...)
	l.buf = append(l.buf, '\n')

	// Если ошибка - нарисуем стек
	// if level == LevelError {

	// 	level := 3
	// 	for {

	// 		_, file, line, ok = runtime.Caller(level)
	// 		if !ok {

	// 			break
	// 		}

	// 		l.buf = append(l.buf, "\t\t"...)
	// 		l.buf = append(l.buf, "Called from "...)
	// 		l.buf = append(l.buf, file...)
	// 		l.buf = append(l.buf, ':')
	// 		itoa(&l.buf, line, -1)
	// 		l.buf = append(l.buf, '\n')

	// 		level++
	// 	}
	// }

	for idx, writer := range l.out {
		if writer != nil {
			if _, err := writer.Write(l.buf); err != nil {
				if err != os.ErrClosed {
					fmt.Printf("log write error: %v\n", err)
				}
				l.out[idx] = nil
			}
		}
	}
}

// log вывести сообщение уровня level
//	* level	- logLevels
//  * s			- ...interface{}
func (l *Logger) log(level logLevels, s ...interface{}) {
	if l.level <= level && len(s) > 0 {
		if first, ok := s[0].(string); ok && strings.Contains(first, "%") && len(s) > 1 {
			l.writeToOut(level, fmt.Sprintf(first, s[1:]...))
		} else {
			l.writeToOut(level, fmt.Sprint(s...))
		}
	}
}

// Print вывести сообщение уровня l.level
//  * s	- ...interface{}
func (l *Logger) Print(s ...interface{}) {
	l.log(l.level, fmt.Sprint(s...))
}

// Verbose вывести сообщение уровня LevelVerbose
//  * s	- ...interface{}
func (l *Logger) Verbose(s ...interface{}) {

	l.log(LevelVerbose, s...)
}

// Debug вывести сообщение уровня LevelDebug
//  * s	- ...interface{}
func (l *Logger) Debug(s ...interface{}) {

	l.log(LevelDebug, s...)
}

// Info вывести сообщение уровня LevelInfo
//  * s	- ...interface{}
func (l *Logger) Info(s ...interface{}) {

	l.log(LevelInfo, s...)
}

// Warn вывести сообщение уровня LevelWarn
//  * s	- ...interface{}
func (l *Logger) Warn(s ...interface{}) {

	l.log(LevelWarn, s...)
}

// Error вывести сообщение уровня LevelError
//  * s	- ...interface{}
func (l *Logger) Error(s ...interface{}) {

	l.log(LevelError, s...)
	// debug.PrintStack()
}

// Fatal вывести сообщение уровня LevelFatal и завершиться
//  * s	- ...interface{}
func (l *Logger) Fatal(s ...interface{}) {

	l.log(LevelFatal, s...)
	Exit(-1)
}

func (l *Logger) FatalGo(s ...interface{}) {
	custom := ""
	_, file, line, ok := runtime.Caller(l.callStackAdder + 1)
	if ok {
		custom = fmt.Sprintf("%v:%v", filepath.Base(file), line)
	}
	go func() {
		l.customFilename = custom
		l.Fatal(s...)
	}()
}

// SetLogLevel устанавливает уровень логгинга
//	* level - logLevels
func (l *Logger) SetLogLevel(level logLevels) {

	l.level = level
}

func (l *Logger) SetCallStackAdder(adder int) {
	l.callStackAdder = adder
}

func (l *Logger) SetNoFileName(no bool) {
	l.noFilename = no
}

// SetWriter устанавливает новый writer для логгера
//	* writer - io.Writer
func (l *Logger) SetWriter(writer io.Writer) {
	l.out[0] = writer
}

func (l *Logger) GetWriter() io.Writer {
	return l.out[0]
}

// AddWriter добавляет writer для логгера
//	* writer - io.Writer
func (l *Logger) AddWriter(writer io.Writer) {
	l.mutex.Lock()
	l.out = append(l.out, writer)
	l.mutex.Unlock()
}

// RemoveWriter удаляет writer логгера
//	* writer - io.Writer
func (l *Logger) RemoveWriter(toRemove io.Writer) (found bool) {
	l.mutex.Lock()
	newLen := 1
	for i := range l.out {
		// Не трогаем 0-го вритера
		if i == 0 {
			continue
		}
		if l.out[i] == toRemove {
			found = true
		} else {
			if i != newLen {
				l.out[newLen] = l.out[i]
			}
			newLen++
		}
	}
	l.out = l.out[0:newLen]
	l.mutex.Unlock()
	return
}

// SetFileWriter устанавливает новый writer для логгера, который пишет в файл
// WARN: автоматически выключает вывод с цветом, чтобы включить - использовать (*Logger)SetUseColors(true)
//	* fileName - string
// Если ошибка - возвращает ошибку, иначе nil
func (l *Logger) SetFileWriter(fileName string) error {

	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {

		return err
	}

	// AtExit(func() {

	// 	f.Close()
	// })

	l.out[0] = f
	l.SetUseColors(false)
	return nil
}

// SetUseColors устанавливает использовать ли цвета при выводе или нет
func (l *Logger) SetUseColors(useColors bool) {

	l.useColors = useColors
}

func AddFileAndLine(err error) error {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return err
	}
	return fmt.Errorf("[%v:%v] %v", filepath.Base(file), line, err)
}

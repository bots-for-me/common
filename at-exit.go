package common

import (
	"os"
	"os/signal"
	"syscall"
	"time"
)

var ExitingChannel = make(chan byte)
var ExitedChannel = make(chan byte)
var ExitWaitChans = NewWaitChans()

var exitInProgress bool

const exitTimeout = time.Second * 15

// Exit - выполняет все запланированные функции и завершает процесс
func Exit(code ...interface{}) {
	if exitInProgress {
		return
	}
	exitInProgress = true
	Log.Info("stopping...")
	close(ExitingChannel)
	time.Sleep(time.Millisecond * 100)

	ExitWaitChans.Wait(exitTimeout)

	exitCode := 0
	needExit := true
	if len(code) == 1 {
		if _exitCode, ok := code[0].(int); ok {
			exitCode = _exitCode
		}
		if _needExit, ok := code[0].(bool); ok {
			needExit = _needExit
		}
	}
	Log.Info("stopped!\n\n")
	close(ExitedChannel)
	time.Sleep(time.Millisecond * 100)
	if needExit {
		os.Exit(exitCode)
	}
}

func init() {
	signalChannel := make(chan os.Signal, 1)
	signal.Ignore(syscall.SIGHUP)
	signal.Notify(signalChannel,
		os.Interrupt,
		syscall.SIGALRM,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	go func() {
		for signal := range signalChannel {
			Log.Warn("Signal %#v received, exiting...", signal.String())
			Exit()
			return
		}
	}()
}

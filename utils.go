package common

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var IsGoRun = false

func init() {
	IsGoRun = strings.Contains(os.Args[0], "go-build")
	rand.Seed(time.Now().UnixNano())
}

func GetCurrentFileAndLine(opts ...int) string {
	depth := 1
	if len(opts) > 0 {
		depth = opts[0]
	}
	_, file, line, _ := runtime.Caller(depth)
	return fmt.Sprintf("%v:%v", filepath.Base(file), line)
}

func Errorf(s ...interface{}) (err error) {
	if len(s) == 0 {
		return
	}
	if first, ok := s[0].(string); ok && len(s) > 1 && strings.Contains(first, "%") {
		// str = fmt.Sprintf(first, s[1:]...)
		s[0] = GetCurrentFileAndLine(2)
		if first[0] == '[' {
			err = fmt.Errorf("[%v]"+first, s...)
		} else {
			err = fmt.Errorf("[%v]: "+first, s...)
		}
	} else if err, ok = s[0].(error); ok && len(s) == 1 {
		if err != nil {
			if err.Error()[0] == '[' {
				err = fmt.Errorf("[%v]%w", GetCurrentFileAndLine(2), err)
			} else {
				err = fmt.Errorf("[%v]: %w", GetCurrentFileAndLine(2), err)
			}
		}
	} else {
		// str = fmt.Sprint(s...)
		err = fmt.Errorf("[%v]: %v", GetCurrentFileAndLine(2), fmt.Sprint(s...))
	}
	return
}

func Atoi(str string, baseArg ...int) (value int) {
	if str == "" {
		return
	}
	base := 10
	if len(baseArg) > 0 {
		base = baseArg[0]
	}
	tmp, err := strconv.ParseInt(str, base, 0)
	if err != nil {
		Log.Warn(err)
	} else {
		value = int(tmp)
	}
	return
}

package common

import (
	"regexp"
	"strings"
	"time"
)

func WatchSrc(ignore ...string) {
	var re *regexp.Regexp
	if len(ignore) > 0 {
		re = regexp.MustCompile("(?:" + strings.Join(ignore, "|") + ")")
	}
	reBadDirs := regexp.MustCompile(`^(?:\.git|lib|bin|dist)`)
	reGoodPath := regexp.MustCompile(`\.(?:go|yaml)$`)
	err := WatchChanges(
		"..",
		func(dir string) bool { return !reBadDirs.MatchString(dir) && (re == nil || !re.MatchString(dir)) },
		func(path string) bool { return reGoodPath.MatchString(path) },
		func(path string) {
			Log.Info("file '%v' changed - exiting", path)
			time.Sleep(time.Second)
			Exit()
		},
	)
	if err != nil {
		Log.Fatal(err)
	}
}

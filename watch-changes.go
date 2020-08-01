package common

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

func WatchChanges(dir string, testDir, testPath func(str string) bool, cb func(path string)) (err error) {
	var reloadWatcher *fsnotify.Watcher
	reloadWatcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			Log.Fatal(err)
		}
		if info.IsDir() && testDir(path) {
			// Log.Verbose("Watch dir '%v'", path)
			if err = reloadWatcher.Add(path); err != nil {
				Log.Fatal(err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	go func() {
		ticker := time.NewTicker(time.Second)
		matched := make(map[string]time.Time, 16)
		for {
			select {
			case event := <-reloadWatcher.Events:
				if testPath(event.Name) {
					Log.Verbose("watch event for '%v': %s", event.Name, event.Op)
					matched[event.Name] = time.Now()
				}
			case err := <-reloadWatcher.Errors:
				if err != nil {
					log.Fatal(err)
				}
			case <-ticker.C:
				for key, ts := range matched {
					if time.Since(ts) > 2*time.Second {
						delete(matched, key)
						Log.Info("watch catched '%v'", key)
						cb(key)
					}
				}
			case <-ExitingChannel:
				ticker.Stop()
				reloadWatcher.Close()
				return
			}
		}
	}()
	return
}

package swagger

import (
	"time"

	"github.com/fsnotify/fsnotify"
)

type WatcherCallback = func()

type Watcher struct {
	watcher      *fsnotify.Watcher
	filename     string
	timer        *time.Timer
	debounceTime time.Duration

	onUpdate chan<- error
	Update   <-chan error
}

const DEFAULT_DEBOUNCE_TIME = 100 * time.Millisecond

func WatchFile(filename string, debounceTime time.Duration) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	updateCh := make(chan error, 1)

	if err := watcher.Add(filename); err != nil {
		if err := watcher.Close(); err != nil {
			panic(err)
		}
		return nil, err
	}

	out := &Watcher{
		watcher:      watcher,
		filename:     filename,
		timer:        nil,
		debounceTime: debounceTime,
		onUpdate:     updateCh,
		Update:       updateCh,
	}

	go out.process()

	return out, nil
}

func (w *Watcher) debounceUpdate() {
	if w.timer != nil {
		w.timer.Stop()
	}

	w.timer = time.AfterFunc(w.debounceTime, func() {
		w.onUpdate <- nil
	})
}

func (w *Watcher) Close() {
	w.watcher.Close()
	w.watcher = nil
}

func (w *Watcher) process() {
	for {
		select {
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.onUpdate <- err
		case ev, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if ev.Has(fsnotify.Write) {
				w.debounceUpdate()
			}
		}
	}
}

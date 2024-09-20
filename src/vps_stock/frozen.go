package vps_stock

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

var notified map[string]time.Time
var mapLock sync.Mutex

func AddNotify(kind string, time time.Time) {
	mapLock.Lock()
	notified[kind] = time
	mapLock.Unlock()
}

func StartFrozen(d time.Duration) {
	notified = make(map[string]time.Time)
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		for {
			select {
			case <-ticker.C:
				mapLock.Lock()
				for k, v := range notified {
					if time.Since(v) > d {
						delete(notified, k)
						log.WithField("product", k).Info("release frozen")
					}
				}
				mapLock.Unlock()
			}
		}
	}()
	return
}

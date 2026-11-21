package deej

import (
	"encoding/json"
	"os"
	"time"
)

type RuntimeStatus struct {
	UpdatedAt time.Time    `json:"updated_at"`
	Version   string       `json:"version"`
	Serial    SerialStatus `json:"serial"`
	Audio     AudioStatus  `json:"audio"`
}

func (d *Deej) writeStatusLoop() {
	defer close(d.statusDone)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	os.MkdirAll("logs", 0755)
	defer os.Remove("logs/status.json")
	for {
		select {
		case <-d.statusStop:
			return
		case <-ticker.C:
			audio := d.sessions.Status()
			_, audio.ConfigError = d.config.State()
			status := RuntimeStatus{time.Now(), d.version, d.serial.Status(), audio}
			if data, err := json.Marshal(status); err == nil {
				if err = atomicWrite("logs/status.json", data); err != nil {
					d.logger.Debugw("Write status", "error", err)
				}
			}
		}
	}
}

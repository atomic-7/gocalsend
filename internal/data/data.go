package data

import (
	"net"
	"sync"
)

type PeerInfo struct {
	Alias       string `json:"alias"`
	Version     string `json:"version"`
	DeviceModel string `json:"deviceModel"` // nullable -> ""
	DeviceType  string `json:"deviceType"`
	// mobile | desktop | web | headless | server | ""
	Fingerprint string `json:"fingerprint"`
	Port        int    `json:"port"`
	Protocol    string `json:"protocol"` // http | https
	Download    bool   `json:"download"` // API > 5.2
	Announce    bool   `json:"announce"`
	IP          net.IP `json:"-"`
}

type PeerMap struct {
	Map  map[string]*PeerInfo
	Lock sync.Mutex
}

func (pm *PeerMap) LockMap() {
	pm.Lock.Lock()
}

func (pm *PeerMap) UnlockMap() {
	pm.Lock.Unlock()
}

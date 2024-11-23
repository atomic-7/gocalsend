package data

import (
	"fmt"
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
	Announce    bool   `json:"announce"` // announce field is on peerinfo because it makes parsing easy, announce can just be checked as a property of the struct this way
	IP          net.IP `json:"-"`
}

func (pi *PeerInfo) ToPeerBody() *PeerBody {
	return &PeerBody{
		Alias:       pi.Alias,
		Version:     pi.Version,
		DeviceModel: pi.DeviceModel,
		DeviceType:  pi.DeviceType,
		Fingerprint: pi.Fingerprint,
		Port:        pi.Port,
		Protocol:    pi.Protocol,
		Download:    false,
	}
}

func (pi *PeerInfo) ToRegisterResponse() *RegisterResponse {
	return &RegisterResponse{
		Alias:       pi.Alias,
		Version:     pi.Version,
		DeviceModel: pi.DeviceModel,
		DeviceType:  pi.DeviceType,
		Fingerprint: pi.Fingerprint,
		Port:        pi.Port,
		Download:    false,
	}
}

func (pi *PeerInfo) ToAnnouncement() *AnnounceInfo {
	return &AnnounceInfo{
		Alias:       pi.Alias,
		Version:     pi.Version,
		DeviceModel: pi.DeviceModel,
		DeviceType:  pi.DeviceType,
		Fingerprint: pi.Fingerprint,
		Port:        pi.Port,
		Protocol:    pi.Protocol,
		Download:    false,
		Announce:    true,
	}
}

type PeerBody struct {
	Alias       string `json:"alias"`
	Version     string `json:"version"`
	DeviceModel string `json:"deviceModel"`
	DeviceType  string `json:"deviceType"`
	// mobile | desktop | web | headless | server | ""
	Fingerprint string `json:"fingerprint"`
	Port        int    `json:"port"`
	Protocol    string `json:"protocol"` // http | https
	Download    bool   `json:"download"` // API > 5.2
}

type RegisterResponse struct {
	Alias       string `json:"alias"`
	Version     string `json:"version"`
	DeviceModel string `json:"deviceModel"`
	DeviceType  string `json:"deviceType"`
	// mobile | desktop | web | headless | server | ""
	Fingerprint string `json:"fingerprint"`
	Port        int    `json:"port"`
	Download    bool   `json:"download"` // API > 5.2
}

type AnnounceInfo struct {
	Alias       string `json:"alias"`
	Version     string `json:"version"`
	DeviceModel string `json:"deviceModel"`
	DeviceType  string `json:"deviceType"`
	// mobile | desktop | web | headless | server | ""
	Fingerprint string `json:"fingerprint"`
	Port        int    `json:"port"`
	Protocol    string `json:"protocol"` // http | https
	Download    bool   `json:"download"` // API > 5.2
	Announce    bool   `json:"announce"`
}

type PeerMap struct {
	peers map[string]*PeerInfo
	lock  sync.Mutex
}

func NewPeerMap() *PeerMap {
	return &PeerMap{
		peers: make(map[string]*PeerInfo),
	}
}

func (pm *PeerMap) GetMap() *map[string]*PeerInfo {
	pm.lock.Lock()
	return &pm.peers
}

func (pm *PeerMap) ReleaseMap() {
	pm.lock.Unlock()
}

type TLSPaths struct {
	Dir  string
	Cert string `toml:",omitempty"`
	Key  string `toml:",omitempty"`
}

func CreateTLSPaths(dir string, certName string, keyName string) *TLSPaths {
	return &TLSPaths{
		Dir:  dir,
		Cert: fmt.Sprintf("%s/%s", dir, certName),
		Key:  fmt.Sprintf("%s/%s", dir, keyName),
	}
}

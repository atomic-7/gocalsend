package main

import (
	"encoding/json"
	"log"
	"net"
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
	IP          net.IP `json:"-"`
	Anounce     bool   `json:"anounce"`
	Anouncement bool   `json:"anouncement"`
}

// The tags in this case are probably optional

func main() {
	iface, err := net.InterfaceByName("wlp3s0")
	if err != nil {
		log.Fatal("Error getting interface: ", err)
	}
	network := "udp4"
	multicastAddr := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 167), Port: 53317}
	mcgroup, err := net.ListenMulticastUDP(network, iface, multicastAddr)
	if err != nil {
		log.Fatal("Error connecting to multicast group: ", err)
	}
	log.Println("gocalsending now!")
	log.Printf("Listening to %s udp multicast group %s:%d", network, multicastAddr.IP.String(), multicastAddr.Port)

	peers := make(map[string]*PeerInfo)

	// hopefully no jumbo frames
	buf := make([]byte, 1500)
	for {
		// consider using mcgroup.ReadMsgUDP
		n, from, err := mcgroup.ReadFromUDP(buf)
		if n != 0 {
			if err != nil {
				log.Fatal("Error reading udp packet", err)
			} else {
				log.Printf("[%s]: %s", from.String(), string(buf))
				info := &PeerInfo{}
				info.IP = from.IP
				err = json.Unmarshal(buf[:n], info) // need to specify the number of bytes read here!
				if err != nil {
					log.Printf("buf: %v", buf[0:400])
					log.Fatal("Error unmarshaling json: ", err)
				}
				log.Printf("alias: %s", info.Alias)
				peers[info.Fingerprint] = info
				log.Printf("struct: %v", *info)

				if info.Anounce {
					log.Printf("Sending node info to %s", info.Alias)
				}
			}
		} else {
			log.Println("Received empty udp packet?")
		}
	}
}

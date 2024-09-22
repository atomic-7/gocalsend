package discovery

import (
	"context"
	"encoding/json"
	"github.com/atomic-7/gocalsend/internal/data"
	"log"
	"net"
)

func AnnounceMulticast(node *data.PeerInfo, multicastAdress *net.UDPAddr) {
	conn, err := net.Dial("udp4", multicastAdress.String())
	if err != nil {
		log.Fatal("Error trying to announce the node via multicast: ", err)
	}
	buf, err := json.Marshal(node.ToAnnouncement())
	if err != nil {
		log.Fatal("Error marshalling node:", err)
	}
	_, err = conn.Write(buf)
	if err != nil {
		log.Fatal("Error announcing node:", err)
	}
}

func MonitorMulticast(ctx context.Context, multicastAddr *net.UDPAddr, peers *data.PeerMap, registratinator *Registratinator) {

	iface, err := net.InterfaceByName("wlp3s0")
	if err != nil {
		log.Fatal("Error getting interface: ", err)
	}
	network := "udp4"
	log.Printf("Listening to %s udp multicast group %s:%d\n", network, multicastAddr.IP.String(), multicastAddr.Port)
	//TODO: rewrite this to manually setup the multicast group to be able to have local packets be visible via loopback
	mcgroup, err := net.ListenMulticastUDP(network, iface, multicastAddr)
	defer mcgroup.Close()
	if err != nil {
		log.Fatal("Error connecting to multicast group: ", err)
	}

	// hopefully no jumbo frames
	buf := make([]byte, 1500)
	for {
		// consider using mcgroup.ReadMsgUDP?
		n, from, err := mcgroup.ReadFromUDP(buf)
		if n != 0 {
			if err != nil {
				log.Fatal("Error reading udp packet", err)
			} else {

				info := &data.PeerInfo{}
				info.IP = from.IP
				err = json.Unmarshal(buf[:n], info) // need to specify the number of bytes read here!
				if err != nil {
					log.Printf("buf: %v", buf[0:400])
					log.Fatal("Error unmarshaling json: ", err)
				}
				log.Printf("[MC][%s]: %s %s", from.String(), info.Alias, info.Protocol)

				pm := *peers.GetMap()
				if _, ok := pm[info.Fingerprint]; !ok {
					log.Printf("[PM]Adding: %s", info.Alias)
					pm[info.Fingerprint] = info
				} else {
					log.Printf("MulticastMonitor: Peer %s was already known", info.Alias)
				}
				peers.ReleaseMap()

				if info.Announce {
					log.Printf("Sending node info to %s", info.Alias)
					err := registratinator.RegisterAt(ctx, info)
					if err != nil {
						log.Println("Pre map lock")
						pm := *peers.GetMap()
						defer peers.ReleaseMap()
						log.Printf("PM: %v\n", pm)
						log.Fatal("Sending node info to peer failed: ", err)
					}
				}
			}
		} else {
			log.Println("Received empty udp packet?")
		}
	}
}

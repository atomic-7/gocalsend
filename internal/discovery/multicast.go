package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/atomic-7/gocalsend/internal/data"
	"log"
	"net"
	"strings"
)

func AnnounceMulticast(node *data.PeerInfo, multicastAdress *net.UDPAddr) {
func AnnounceViaMulticast(node *data.PeerInfo, multicastAdress *net.UDPAddr) {
	conn, err := net.Dial("udp4", multicastAdress.String())
	if err != nil {
		log.Fatal("Error trying to announce the node via multicast: ", err)
	}
	buf, err := json.Marshal(node.ToAnnouncement())
	if err != nil {
		log.Fatal("Error marshalling node: ", err)
	}
	_, err = conn.Write(buf)
	if err != nil {
		log.Fatal("Error writing node info: ", err)
	}
}

func MonitorMulticast(ctx context.Context, multicastAddr *net.UDPAddr, peers *data.PeerMap, registratinator *Registratinator) {

	ifaces, err := net.Interfaces()
	if err != nil {
		log.Fatal("Failed getting list of interfaces: ", err)
	}
	candidates := make([]*net.Interface, 0, len(ifaces))
	log.Println("Setting up multicast interface")
	for _, ife := range ifaces {
		
		if strings.Contains(ife.Name, "lo") {
			// TODO: Improve loopback interface detection
			log.Println("[lo] Skip(loopback)")
			continue
		}
		if strings.Contains(ife.Name, "docker") {
			log.Printf("[%s] Skip(Docker)\n", ife.Name)
			continue
		}
		if !hasFlag(ife, net.FlagUp) {
			log.Printf("[%s] Skip(Interface down)\n", ife.Name)
			continue
		}
		if !hasFlag(ife, net.FlagRunning) {
			log.Printf("[%s] Skip(Not running)\n", ife.Name)
			continue
		}
		if !hasFlag(ife, net.FlagMulticast) {
			log.Printf("[%s] Skip(No multicast)\n", ife.Name)
		}
		candidates = append(candidates, &ife)
	}

	switch len(candidates) {
	case 0:
		log.Fatal("Found no viable interface for multicast")
	case 1:
		log.Println("Found one viable network interface")
	default:
		log.Printf("Found %d viable network interfaces", len(candidates))
		for _, ife := range candidates {
			fmt.Printf("[%s] %s\n", ife.Name, ife.Flags.String())
		}
	}
	iface := candidates[0]
	log.Printf("Selecting %s", iface.Name)
	// iface, err := net.InterfaceByName("wlp3s0")
	network := "udp4"
	log.Printf("Listening to %s multicast group %s:%d\n", network, multicastAddr.IP.String(), multicastAddr.Port)
	//TODO: rewrite this to manually setup the multicast group to be able to have local packets be visible via loopback
	mcgroup, err := net.ListenMulticastUDP(network, iface, multicastAddr)
	defer mcgroup.Close()
	if err != nil {
		log.Fatal("Error connecting to multicast group: ", err)
	}

	buf := make([]byte, iface.MTU)
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

				unknownPeer := true
				pm := *peers.GetMap()
				if _, ok := pm[info.Fingerprint]; !ok {
					log.Printf("[PM]Adding: %s", info.Alias)
					pm[info.Fingerprint] = info
				} else {
					unknownPeer = false
					log.Printf("[MC]: Peer %s was already known", info.Alias)
				}
				peers.ReleaseMap()

				if info.Announce && unknownPeer {
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

func hasFlag(iface net.Interface, flag net.Flags) bool {
	for ifFlag := range iface.Flags {
		if ifFlag == flag {
			return true
		}
	}
	return false
}

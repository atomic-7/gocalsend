package main

import (
	"log"
	"net"
)

func main() {
	iface, err := net.InterfaceByName("wlp3s0")
	if err != nil {
		log.Fatal("Error getting interface:", err)
	}
	network := "udp4"
	multicastAddr := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 0), Port: 53317}
	mcgroup, err := net.ListenMulticastUDP(network, iface, multicastAddr)
	if err != nil {
		log.Fatal("Error connecting to multicast group", err)
	}
	log.Println("gocalsending now!")
	log.Printf("Listening to %s udp multicast group %s:%d", network, multicastAddr.IP.String(), multicastAddr.Port)

	buf := make([]byte, 1500)
	for {
		// consider using mcgroup.ReadMsgUDP
		n, from, err := mcgroup.ReadFromUDP(buf)
		if n != 0 {
			// maybe append bytes until some kind of error?
			if err != nil {
				log.Fatal("Error reading udp packet", err)
			} else {
				log.Printf("[%s]: %s", from.String(), string(buf))
			}
		} else {
			log.Println("Received empty udp packet?")
		}
	}
}

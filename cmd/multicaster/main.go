package main

import (
	"log"
	"net"
)

func main() {
	multicast := &net.UDPAddr{IP: net.IPv4(240, 0, 0, 167), Port: 53317}
	network := "udp4"
	conn, err := net.DialUDP(network, nil, multicast)
	if err != nil {
		log.Fatal("Error dialing to multicast", err)
	}
	log.Println("Multicasting at the speed of light!")
	conn.Write([]byte("Woo!"))
	log.Println("Multicast gone, carry on!")
}

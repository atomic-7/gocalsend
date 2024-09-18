package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/atomic-7/gocalsend/internal/data"
	"github.com/atomic-7/gocalsend/internal/encryption"
	"github.com/atomic-7/gocalsend/internal/server"
	"io"
	"log"
	"net"
	"net/http"
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

// Send a post request to /api/localsend/v2/register with the node data
func SendTo(ctx context.Context, peer *data.PeerInfo, nodeJson []byte) error {
	log.Println("Called SendTo")

	url := fmt.Sprintf("http://%s:%d/api/localsend/v2/register", peer.IP, peer.Port)
	//url := "http://localhost:8123/api/localsend/v2/register"
	log.Printf("Using: %s with %s", url, string(nodeJson))

	req, err := http.NewRequest("post", url, bytes.NewReader(nodeJson))
	if err != nil {
		log.Fatal("Error creating post request to %s", url)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", "go1.23.0 linux/amd64")
	// req.Header.Add("Encoding", "identity")
	req.Close = true
	resp, err := http.DefaultClient.Do(req)

	// These cannot be logged as the response is an invalid pointer here due to the error
	// log.Printf("RespStatus: %s", resp.Status)
	// log.Printf("RespHeaders: %v", resp.Header)

	// resp, err := http.Post(url, "application/json", bytes.NewReader(nodeJson))
	if err != nil {
		if errors.Is(err, io.EOF) {
			// Sending node info to peer failed: Post "http://192.168.117.39:53317/api/localsend/v2/register": EOF
			log.Println("How the hell did we get here?")
		}
		return err
	}
	// don't know if the response is  going to be interesting
	log.Printf("Response to post req should be %d bytes", resp.ContentLength)
	defer resp.Body.Close()
	if resp.ContentLength != 0 {

		body, err := io.ReadAll(resp.Body)
		if !errors.Is(err, io.EOF) {
			return err
		}

		log.Println("%s responds with %s", peer.Alias, body)
	}
	log.Println("Sent off local node info!")
	return nil
}

func MonitorMulticast(ctx context.Context, multicastAddr *net.UDPAddr, peers *data.PeerMap, jsonBuf []byte) {

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

				log.Printf("[MC][%s]: %s ...", from.String(), string(buf[0:50]))
				info := &data.PeerInfo{}
				info.IP = from.IP
				err = json.Unmarshal(buf[:n], info) // need to specify the number of bytes read here!
				if err != nil {
					log.Printf("buf: %v", buf[0:400])
					log.Fatal("Error unmarshaling json: ", err)
				}

				peers.Lock.Lock()
				if _, ok := peers.Map[info.Fingerprint]; !ok {
					log.Printf("[PM]Adding: %s", info.Alias)
					peers.Map[info.Fingerprint] = info
				} else {
					log.Printf("MulticastMonitor: Peer %s was already known", info.Alias)
				}
				peers.Lock.Unlock()

				if info.Announce {
					log.Printf("Sending node info to %s", info.Alias)
					err := SendTo(ctx, info, jsonBuf)
					if err != nil {
						log.Printf("PM: %v\n", peers.Map)
						log.Fatal("Sending node info to peer failed: ", err)
					}
				}
			}
		} else {
			log.Println("Received empty udp packet?")
		}
	}
}
func main() {

	var port int
	var certPath string
	var keyPath string

	flag.IntVar(&port, "port", 53317, "The port to listen for the api endpoints")
	flag.StringVar(&certPath, "certpath", "cert/cert.pem", "The path to the tls certificate")
	flag.StringVar(&keyPath, "keypath", "cert/key.pem", "The path to the tls private key")

	tlsInfo := &data.TLSPaths{
		Cert:       certPath,
		PrivateKey: keyPath,
	}

	// TODO: Read cert, make sha256 and set to fingerprint
	node := &data.PeerInfo{
		Alias:       "Gocalsend",
		Version:     "2.0",
		DeviceModel: "cli",
		DeviceType:  "server",
		Fingerprint: encryption.GetFingerprint(certPath),
		Port:        port,
		Protocol:    "https", // changing this to https might work to prevent the other client from going with the info route
		Download:    false,
		IP:          nil,
		Announce:    false,
	}
	peers := &data.PeerMap{Map: make(map[string]*data.PeerInfo)}
	jsonBuf, err := json.Marshal(node.ToPeerBody())
	if err != nil {
		log.Fatal("Error marshalling local node to json: ", err)
	}

	multicastAddr := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 167), Port: 53317}
	// When we multicast first, registry via our http endpoint is fine. Me calling their endpoint results in a crash
	// AnnounceMulticast(node, multicastAddr)
	log.Println("gocalsending now!")

	ctx,cancel := context.WithCancel(context.Background())
	defer cancel()
	go server.StartServer(ctx, fmt.Sprintf(":%d", node.Port),fmt.Sprintf(":%d", node.Port + 1), peers, tlsInfo)
	MonitorMulticast(ctx, multicastAddr, peers, jsonBuf)
}

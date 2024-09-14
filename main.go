package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
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

func (node *PeerInfo) AnnounceMulticast(multicastAdress *net.UDPAddr) {
	conn, err := net.Dial("udp4", multicastAdress.String())
	if err != nil {
		log.Fatal("Error trying to announce the node via multicast: ", err)
	}
	node.Announce = true
	buf, err := json.Marshal(node)
	if err != nil {
		log.Fatal("Error marshalling node:", err)
	}
	_, err = conn.Write(buf)
	if err != nil {
		log.Fatal("Error announcing node:", err)
	}
}

// The tags in this case are probably optional
func SendTo(ctx context.Context, peer *PeerInfo, nodeJson []byte) error {
	log.Println("Called SendTo")

	url := fmt.Sprintf("http://%s:%d/api/localsend/v2/register", peer.IP, peer.Port)
	log.Printf("Using: %s with %s", url, string(nodeJson))

	resp, err := http.Post(url, "application/json", bytes.NewReader(nodeJson))
	if err != nil {
		return err
	}
	// don't know if the response is  going to be interesting
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	// Sending node info to peer failed: Post "http://192.168.117.39:53317/api/localsend/v2/register": EOF
	if !errors.Is(err, io.EOF) {
		return err
	}

	log.Println("%s responds with %s", peer.Alias, body)
	return nil
}

func MonitorMulticast(ctx context.Context, multicastAddr *net.UDPAddr, peers *PeerMap, jsonBuf []byte) {

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
				log.Printf("[%s]: %s", from.String(), string(buf))
				info := &PeerInfo{}
				info.IP = from.IP
				err = json.Unmarshal(buf[:n], info) // need to specify the number of bytes read here!
				if err != nil {
					log.Printf("buf: %v", buf[0:400])
					log.Fatal("Error unmarshaling json: ", err)
				}
				log.Printf("alias: %s", info.Alias)
				peers.Lock.Lock()
				if _, ok := peers.Map[info.Fingerprint]; !ok {
					log.Printf("Adding: %v", *info)
					peers.Map[info.Fingerprint] = info
				}
				peers.Lock.Unlock()

				if info.Announce {
					log.Printf("Sending node info to %s", info.Alias)
					err := SendTo(ctx, info, jsonBuf)
					if err != nil {
						log.Fatal("Sending node info to peer failed: ", err)
					}
				}
			}
		} else {
			log.Println("Received empty udp packet?")
		}
	}
}

func CreateRegisterHandler(peers *PeerMap) http.Handler {

	return http.HandlerFunc(func (writer http.ResponseWriter, r *http.Request) {
		buf, err := io.ReadAll(r.Body)
		if err != nil {
			log.Fatal("Error reading register body: ", err)
		}
		var peer PeerInfo
		json.Unmarshal(buf, &peer)
		log.Printf("Registering %s via http register route", peer.Alias) // remember to lock the mutex
		peers.LockMap()
		defer peers.UnlockMap()
		if _,ok := peers.Map[peer.Fingerprint]; ok {
			log.Printf("%s was already a known peer", peer.Alias)
		} else {
			peers.Map[peer.Fingerprint] = &peer	
		}
	})
}
func RunServer(port string, peers *PeerMap) {
	
	if peers == nil {
		log.Fatal("error setting up server, peermap is nil")
	}

	mux := http.NewServeMux()
	mux.Handle("/api/localsend/v2/register", CreateRegisterHandler(peers))
	
	err := http.ListenAndServe(port, mux)
	log.Fatalf("Error running server at port %s: %s", port, err)
}

func main() {

	node := &PeerInfo{
		Alias:       "Gocalsend",
		Version:     "2.0",
		DeviceModel: "cli",
		DeviceType:  "server",
		Fingerprint: "nonononono",
		Port:        53317,
		Protocol:    "http",
		Download:    false,
		IP:          nil,
		Announce:    false,
	}
	peers := &PeerMap{Map: make(map[string]*PeerInfo)}
	node.Announce = false
	jsonBuf, err := json.Marshal(node)
	if err != nil {
		log.Fatal("Error marshalling local node to json: ", err)
	}

	multicastAddr := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 167), Port: 53317}
	node.AnnounceMulticast(multicastAddr)
	log.Println("gocalsending now!")

	ctx := context.Background()
	go RunServer(fmt.Sprintf(":%d", node.Port), peers)
	MonitorMulticast(ctx, multicastAddr, peers, jsonBuf)

}

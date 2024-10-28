package discovery

import (
	"context"
	"encoding/json"
	"github.com/atomic-7/gocalsend/internal/data"
	"log/slog"
	"net"
	"os"
	"strings"
)

// Fallback registration method in case the register endpoint does not work
func RegisterViaMulticast(node *data.PeerInfo, multicastAdress *net.UDPAddr) {
	conn, err := net.Dial("udp4", multicastAdress.String())
	if err != nil {
		slog.Error("failed to register the node via multicast", slog.Any("error", err))
		os.Exit(1)
	}
	registration := node.ToAnnouncement()
	registration.Announce = false
	buf, err := json.Marshal(registration)
	if err != nil {
		slog.Error("Error marshalling node", slog.Any("error", err))
		os.Exit(1)
	}
	_, err = conn.Write(buf)
	if err != nil {
		slog.Error("Error writing node info", slog.Any("error", err))
		os.Exit(1)
	}
}

// Blast node info to the multicast address
func AnnounceViaMulticast(node *data.PeerInfo, multicastAdress *net.UDPAddr) error {
	conn, err := net.Dial("udp4", multicastAdress.String())
	slog.Debug("announcing via multicast", slog.String("addr", multicastAdress.String()))
	if err != nil {
		slog.Error("Error trying to announce the node via multicast", slog.Any("error", err))
		os.Exit(1)
	}
	buf, err := json.Marshal(node.ToAnnouncement())
	if err != nil {
		slog.Error("Error marshalling node: ", err)
	}
	_, err = conn.Write(buf)
	if err != nil {
		slog.Error("Error writing node info", slog.Any("error", err))
		return err
	}
	return nil
}

func MonitorMulticast(ctx context.Context, multicastAddr *net.UDPAddr, peers *data.PeerMap, registratinator *Registratinator) {

	iface := GetInterface()
	slog.Debug("interface setup", slog.String("interface", iface.Name))
	network := "udp4"
	slog.Debug("listening to multicast group", slog.String("network", network), slog.String("ip", multicastAddr.IP.String()), slog.Int("port", multicastAddr.Port))
	//TODO: rewrite this to manually setup the multicast group to be able to have local packets be visible via loopback
	mcgroup, err := net.ListenMulticastUDP(network, iface, multicastAddr)
	defer mcgroup.Close()
	if err != nil {
		slog.Error("Error connecting to multicast group", slog.Any("error", err))
		os.Exit(1)
	}

	buf := make([]byte, iface.MTU)
	for {
		// consider using mcgroup.ReadMsgUDP?
		n, from, err := mcgroup.ReadFromUDP(buf)
		if n != 0 {
			if err != nil {
				slog.Error("Error reading udp packet", slog.Any("error", err))
				os.Exit(1)
			} else {
				info := &data.PeerInfo{}
				info.IP = from.IP
				err = json.Unmarshal(buf[:n], info) // need to specify the number of bytes read here!
				if err != nil {
					slog.Debug("raw udp packet", slog.Any("buf", buf[0:400]))
					slog.Error("failed to unmarshal json", slog.Any("error", err))
					continue
				}
				slog.Debug("multicast discovery", slog.String("ip", from.String()), slog.String("alias", info.Alias), slog.String("protocol", info.Protocol))

				pm := *peers.GetMap()
				if pm["self"].Fingerprint == info.Fingerprint {
					continue
				}
				if _, ok := pm[info.Fingerprint]; !ok {
					slog.Info("adding peer", slog.String("peer", info.Alias), slog.String("source", "multicast"))
					pm[info.Fingerprint] = info
				} else {
					slog.Info("received advertisement from known peer", slog.String("peer", info.Alias))
				}
				peers.ReleaseMap()

				if info.Announce {
					// TODO: delay this. I am currently sniping a starting instance before the http server is up
					slog.Info("sending local node info", slog.String("peer", info.Alias))
					err := registratinator.RegisterAt(ctx, info)
					if err != nil {
						slog.Error("failed to send node info to peer", slog.String("peer", info.Alias), slog.Any("error", err))
						RegisterViaMulticast(info, multicastAddr)
					}
				} else {
					slog.Info("incoming registry via multicast fallback", slog.String("peer", info.Alias), slog.String("source", "multicast"))
				}
			}
		} else {
			slog.Debug("received empty udp packet?")
		}
	}
}

// return the network interface. kills the program if none can be found
func GetInterface() *net.Interface {

	ifaces, err := net.Interfaces()
	if err != nil {
		slog.Error("Failed getting list of interfaces", slog.Any("error", err))
		os.Exit(1)
	}
	candidates := make([]*net.Interface, 0, len(ifaces))
	slog.Debug("setting up multicast interface")
	for _, ife := range ifaces {

		if strings.Contains(ife.Name, "lo") {
			// TODO: Improve loopback interface detection
			slog.Debug("skipping interface", slog.String("interface", ife.Name), slog.String("reason", "loopback"))
			continue
		}
		if strings.Contains(ife.Name, "docker") {
			slog.Debug("skipping interface", slog.String("interface", ife.Name), slog.String("reason", "docker"))
			continue
		}
		if ife.Flags&net.FlagUp == 0 {
			slog.Debug("skipping interface", slog.String("interface", ife.Name), slog.String("reason", "down"))
			continue
		}
		if ife.Flags&net.FlagRunning == 0 {
			slog.Debug("skipping interface", slog.String("interface", ife.Name), slog.String("reason", "not running"))
			continue
		}
		if ife.Flags&net.FlagRunning == 0 {
			slog.Debug("skipping interface", slog.String("interface", ife.Name), slog.String("reason", "no multicast"))
		}
		candidates = append(candidates, &ife)
	}

	switch len(candidates) {
	case 0:
		slog.Error("found no viable interface for multicast")
		os.Exit(1)
	case 1:
		slog.Debug("found one viable network interface", slog.String("interface", candidates[0].Name))
	default:
		slog.Debug("found multiple viable network interfaces", slog.Int("num", len(candidates)))
		for _, ife := range candidates {
			slog.Debug("interface candidate", slog.String("interface", ife.Name), slog.String("flags", ife.Flags.String()))
		}
	}
	// copy out the struct so not the entire slice needs to be kept allocated
	iface := *candidates[0]
	return &iface
}

# Notes

## Networking
The godocs of the "net" package have a function ListenMulticastUDP.
They also mention https://pkg.go.dev/golang.org/x/net/ipv4 for more involved use cases, might have to take a look at this

## Configuration
Just the same as the gui version, settings are stored in `settings.json` 
## Protocol
- Incoming: Port 53317
### Multicast
- Default multicast group is 224.0.0.0/24
- multicast group on my phone is 224.0.0.167/24
- default multicast port: 53317
Multicast only targets clients that have subscribed to a multicast group as opposed to a 
broadcast that is sent to every client in the network
[go-multicast](https://github.com/dmichael/go-multicast)

### Fingerprint
https: With encryption the fingerprint is the sh256 hash of the certificate
http: Without encryption the fingerprint is a random generated string

### Discovery
At start the node sends to multicast group:
{
  "alias": "Nice Orange",
  "version": "2.0", // protocol version (major.minor)
  "deviceModel": "Samsung", // nullable
  "deviceType": "server", // mobile | desktop | web | headless | server, nullable, check what the other clis set here
  "fingerprint": "random string",	// only used if not encrypted
  "port": 53317,
  "protocol": "https", // http | https
  "download": true, // if the download API (5.2 and 5.3) is active (optional, default: false)
  "announce": true
}
The peers respond with
POST /api/localsend/v2/register

{
  "alias": "Secret Banana",
  "version": "2.0",
  "deviceModel": "Windows",
  "deviceType": "desktop",
  "fingerprint": "random string", // ignored in HTTPS mode
  "port": 53317,
  "protocol": "https",
  "download": true, // if the download API (5.2 and 5.3) is active (optional, default: false)
}

As fallback, members can also respond with a Multicast/UDP message.
{
  "alias": "Secret Banana",
  "version": "2.0",
  "deviceModel": "Windows",
  "deviceType": "desktop",
  "fingerprint": "random string",
  "port": 53317,
  "protocol": "https",
  "download": true,
  "announce": false,
}

Legacy mode via http:
If multicast was unsuccessful this is sent to all local ip adresses
POST /api/localsend/v2/register
{
  "alias": "Secret Banana",
  "version": "2.0", // protocol version (major.minor)
  "deviceModel": "Windows",
  "deviceType": "desktop",
  "fingerprint": "random string", // ignored in HTTPS mode
  "port": 53317,
  "protocol": "https", // http | https
  "download": true, // if the download API (5.2 and 5.3) is active (optional, default: false)
}
response:
{
  "alias": "Nice Orange",
  "version": "2.0",
  "deviceModel": "Samsung",
  "deviceType": "mobile",
  "fingerprint": "random string", // ignored in HTTPS mode
  "download": true, // if the download API (5.2 and 5.3) is active (optional, default: false)
}


For some reason the mobile client does not seem to support the register endpoint when encryption is on. The desktop client shows this in the logs:
`[INFO] [Multicast] Respond to announcement of Smart Cookie (192.168.117.39, model: Honor) with UDP because TCP failed`
This curl call seems to yield the correct response however:
` //curl --json '{"alias":"Gocalsend","version":"2.0","deviceModel":"cli","deviceType":"headless","fingerprint":"3d7b158a3f1279bab4c1926b1375bfbd05af954dbaaef7e4ff3ead226dbe9288","port":53320,"protocol":"https","download":false}' --insecure https://192.168.117.39:53317/api/localsend/v2/register`
This could be because the endpoint might still be http only and curl ignores that with --insecure??
But then it should work to just send the request to the register endpoint via normal http.
Does not work tho


### TLS
use
`openssl s_client -connect <ip:port> | openssl x509 -text -noout`
to get info on the certificate used by localsend. There is also a debugging page in the about section found in the settings
The reference implementation seems to use a 2048 bit rsa private key with sha256 for their signature.
They do not seem to set any subject alternative names (san) which is going to cause issues with the standard tls implementation of golang.
They also just use 1 as the serial number.
There also do not seem to be any x509v3 extensions used

# Notes

## Configuration
Just the same as the gui version, settings are stored in `settings.json` 
## Protocol
- Incoming: Port 53317
### Multicast
- Default multicast group is 224.0.0.0/24
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

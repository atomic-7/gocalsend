# TODO

- [] Advertise local node to peers
     - [x] send advertisement to multicast address via udp
     - [] send own encryption key to peers (should be in advertisement as fingerprint)
     - [] store answers in fingerprint->peer map
- [] Discover peers
     - [x] listen on multicast address, make sure to enable loopback so local testing becomes possible
     - [] receive a peers encryption keys (fingerprint), add to certpool?
     - [] hit their /api/localsend/v2/register via https
     - [] host own /api/localsend/v2/register endpoint so peers can answer to the advertisement

- [] Encryption
	- [] generate a certificate -> https://eli.thegreenplace.net/2021/go-https-servers-with-tls/
	- [] encrypt a file using the certificate
    - [] decrypt a file using a peers certificate <- should be transparent using https
    encryption and decryption should just be tls

- [] Protocol parsing
    - [] support version 1
    - [] support version 2 (https only)

- [] Session manager
    - [] map between fingerprints and peers with a mutex

- [] Receive a single file
    - start http server, listen at /api/localsend/v2/prepare-upload
    - create session and register it with session manager
    - respond with session id and file tokens
    - receive file at /api/localsend/v2/upload?sessionId=<id>&fileId=<fileid>&token=<fileToken>
        (upload route should be callable in parallel)
- [] Receive multiple files
	- [] File sink: maintain a list of received files
- [] Send cmdline arg text
- [] Send a single file
    - send post request to target/api/localsend/v2/prepare-upload
        {"info":"<local node info>", "files": { "some-file-id":{..}, "other-file-id":{}}}
    - recieve session id and file tokens as a response
    - send post request to target/api/localsend/v2/upload?sessionId=<id>&fileId=<fileid>&token=<fileToken>
- [] Send multiple files

- [] cancel session
    - [] implement /api/localsend/v2/cancel?sessionId="<sessionId>"

- [] Reverse File transfer for when localsend is not available on the client
- [] pin support
- [] TUI with the charm libraries


## CLI (shortcut to gclsnd?)
´gclsnd ls`
`gocalsend list`
- list all currently online peers
- save previously unknown peers to known peers db

´gclsnd rec`
´gocalsend receive [--out <some-path>]`
- accept any inbound file transfers if there is space available in path
- save the inbound files to path
- save to cwd if no path given

´gclsnd snd <target> -- "sometext" <file1> <file2>`
- offer a filetransfer to target, transmit files if accepted
- return value != 0 if transfer not accepted

`gclsnd ls peers`
- list all peers from the database


## Extra functionality
- [] Blaster mode: Send a copy of a file to any client that connects to the peer
- [] Send queue: enque files for when a known peer becomes available again
- [] compat mode: once the official client releases there should be a config option to be compatible (drop in solution)

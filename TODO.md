# TODO

- [x] Advertise local node to peers
     - [x] send advertisement to multicast address via udp
     - [x] store answers in fingerprint->peer map
- [x] Discover peers
     - [x] listen on multicast address, make sure to enable loopback so local testing becomes possible
     - [x] hit their /api/localsend/v2/register via https
     - [x] host own /api/localsend/v2/register endpoint so peers can answer to the advertisement

- [] Encryption
	- [x] generate a certificate -> https://eli.thegreenplace.net/2021/go-https-servers-with-tls/
	- [x] encrypt a file using the certificate
	- [] Try to hook into the tls handshake and see if the peer cert can be added to the trusted pool if the sha256 of the cert matches the fingerprint
	- [] track the localsend mtls state, it is not supported yet in the official client so no need to worry yet
- [] Protocol parsing
    - [] support version 1 (not a priority)
    - [x] support version 2 

- [] Session manager
    - [x] map between fingerprints and peers with a mutex
    - [] track which sessions belong to which peer for added security
    - [] pin validation

- [x] Receive a single file
    - [x] start http server, listen at /api/localsend/v2/prepare-upload
    - [x] create session and register it with session manager
    - [x] respond with session id and file tokens
    - [x] receive file at /api/localsend/v2/upload?sessionId=<id>&fileId=<fileid>&token=<fileToken>
        (upload route should be callable in parallel)
- [x] Receive multiple files
	- [x] File sink: maintain a list of received files (session manager)
- [x] Send a single file
    - [x] send post request to target/api/localsend/v2/prepare-upload
        {"info":"<local node info>", "files": { "some-file-id":{..}, "other-file-id":{}}}
    - [x] recieve session id and file tokens as a response
    - [x] send post request to target/api/localsend/v2/upload?sessionId=<id>&fileId=<fileid>&token=<fileToken>
- [] Send cmdline arg text
- [] Send multiple files
- [] Improve argument parsing, could use flag groups
    -> switch on the firsts argument, then parse the corresponding flag group

- [] cancel session
    - [x] implement /api/localsend/v2/cancel?sessionId="<sessionId>"
	- the reference client does not seem to send a sessionId?
    - [] maybe try to get hold of currently active transfers belonging to the session and cancel

- [] Reverse File transfer for when localsend is not available on the client
- [] pin support
- [] TUI with the charm libraries
    - [] stack based scene manager to be able to go back easily? could be useful to return after incoming session requests
    - [] split component that displays the help view of the active component on the bottom as well as an indicator for the active component?
    - [] fancy lipgloss styles
    - [x] add a back button so the user can correct selected files
    - [] multi file select
	- [x] allow deselecting files in the selected file list
	- [] make selecting folders configurable?
    - [] settings screen, write changes to the config file
	- [] figure out the best way to update the config of the running program (restarting all components would work I guess but is kinda lame)
	- [] maybe reconsider the eventbus thing to use it for config updates
    - [x] screen to select peers
	- [x] display selected files
	- [] refresh available peers on peer screen
    - [x] grab focus for session offers
- [] Progress display
    - [] provide hooks to the upload handler so it can report the progress to the ui?

- [x] Event Bus to decouple events from UI
    - [x] decided against doing it because the ui hook solution already decouples the headless client sufficiently and handles session offers neatly
- [] Logging
    - [x] Look into structural logging with slog
    - [x] Add log levels for debugging
    - [] Implement LogValuer interface for some structs ?
    - [x] see if the charm logger can work with slog (it should)
    - [] make an enum for exit codes that can be used instead of log fatal

- [] Testing
    - [] End to End tests for two clients on the same machine
	- communication with ref peers seems to work but not with gocalsend peers?
	- works between different machines
    - [] Figure out unit testing for http endpoints

- [] Config
    - [x] Choose a suitable config format
    - [x] use config to allow user to specify their own tls certs

- [] Misc
    - [] generate a random fingerprint
    - [] translate ~ to the correct user home for all paths
    - [] random name from hostname/user/word combo
    

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

### Interactive inline mode
- sending is only really viable when peers are known
- present list of peers, take user input and then send files:
´gclsnd snd <file1> <file2> <file3>´
Enter number to send to peer:
[1] Alias1
[2] Alias2
[3] Alias3
[4] Alias4


## Extra functionality
- [] Blaster mode: Send a copy of a file to any client that connects to the peer
- [] Send queue: enque files for when a known peer becomes available again
- [] compat mode: once the official client releases there should be a config option to be compatible (drop in solution)

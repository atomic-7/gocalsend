# Gocalsend
An implementation of the localsend protocol in go.

## Usage
### Discover Peers
Use `gocalsend --cmd=ls` to discover peers.
```
gocalsend --cmd=ls --lstime=4
```
The `lstime` parameter is optional and specifies how long to wait for peers to respond.
### Send a File
After having found the alias of the peer you want to send your files to, use `gocalsend --cmd=send` to send them.
```
gocalsend --cmd=send --peer=<your peer alias here> <file1> <file2> <file3>
```
Instead of `--cmd=send` you could also use the short form '--cmd=snd'. Gotta save those characters.
### Receive Files
Use `gocalsend --cmd=receive` to wait for incoming files from peers on the network.
```
gocalsend --cmd=receive
```
Gocalsend accepts all incoming sessions right now and saves their files to the base download folder. To change where the files go, use the `--out=<path>` flag. If you need gocalsend to use a different port use the `--port=1234` flag. The default port is 53317 as used by the localsend reference implementation.
```
gocalsend --cmd=receive --out=<path> --port=53320
```
Instead of `receive` you may also pass `rcv`, `rec` or `recv` to save some time.
If you ***really*** want to save time then don't pass any command, to receive files is the default behavior.
### Encryption
Per default, gocalsend creates a subfolder `./cert` in the folder it is currently running.
If this folder already exists it will reuse the certificate inside, otherwise a new one will be generated. If you want to use your own certificate you will have to specify where the cert and its key are stored.
```
gocalsend --receive --cert=<path/to/cert> --key=<path/to/key>
```
Hopefully you will be able to do this in the config someday.
gocalsend uses a rsa 2048 bit privte key as that is what the localsend reference implementation does.

### Logging
The log level can be set to one of either `none`, `debug` or `info`. 
```
gocalsend --loglevel=debug
```
The `none` log level will be implemented soon<sup>TM</sup>

## Building
Run for a release build:
```
make release
```
Run for a development build with debug symbols and a bunch of odd test binaries:
```
make all
```

## Related work
- [protocol](https://github.com/localsend/protocol) Protocol documentation
### Libraries
- [charmbracelet/log](https://github.com/charmbracelet/log) A slog compatible pretty logger
### Alternative implementations
If your implementation is missing here, let me know
- [localsend](https://github.com/localsend/localsend) The reference implementation
- [localsend-rs](https://github.com/zpp0196/localsend-rs) Awesome rust implementation
- [go-localsend](https://github.com/meowrain/localsend-go) An alternative go implementation, check the docs folder for the english readme


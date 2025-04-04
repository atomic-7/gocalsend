# Gocalsend
An implementation of the localsend protocol in go.

There is a headless client called `gclsnd` and the tui client called `gocalsend`
Currently only the headless client allows using the `--cmd` flag.

## Usage
### Discover Peers
Use `gocalsend --cmd=ls` to discover peers.
```
gclsnd --cmd=ls --lstime=4
```
The `lstime` parameter is optional and specifies how long to wait for peers to respond.
### Send a File
After having found the alias of the peer you want to send your files to, use `gocalsend --cmd=send` to send them.
```
gclsnd --cmd=send --peer=<your peer alias here> <file1> <file2> <file3>
```
Instead of `--cmd=send` you could also use the short form '--cmd=snd'. Gotta save those characters.
### Receive Files
Use `gocalsend --cmd=receive` to wait for incoming files from peers on the network.
```
gclsnd --cmd=receive
```
Gocalsend accepts all incoming sessions right now and saves their files to the base download folder. To change where the files go, use the `--out=<path>` flag. If you need gocalsend to use a different port use the `--port=1234` flag. The default port is 53317 as used by the localsend reference implementation.
```
gclsnd --cmd=receive --out=<path> --port=53320
```
Instead of `receive` you may also pass `rcv`, `rec` or `recv` to save some time.
If you ***really*** want to save time then don't pass any command, to receive files is the default behavior.
### Encryption
Per default, gocalsend creates a subfolder `./cert` in the folder it is currently running.
If this folder already exists it will reuse the certificate inside, otherwise a new one will be generated. If you want to use your own certificate you will have to specify where the cert and its key are stored.
```
gclsnd --cert=<path/to/cert> --key=<path/to/key>
```
Hopefully you will be able to do this in the config someday.
gocalsend uses a rsa 2048 bit privte key as that is what the localsend reference implementation does.

### Logging
The log level can be set to one of either `none`, `debug` or `info`. 
```
gocalsend --loglevel=debug
```
The `none` log level will be implemented soon<sup>TM</sup>

## Configuration
The configuration file is stored in the xdg config folder. If this environment variable is unset this location defaults to $HOME/.config/ on unix based systems or %appdata% on windows. 
When first running gocalsend the program will create a new folder called `gocalsend` in this directory. Within that folder you will find a file called `config.toml` as well as the tls certificate and key. The keys of the configuration file are not yet final, which is indicated by the version number 0 in the config file. Command line flags always take precedence over values from the config file. The `--config` flag can be used to supply a different configuration.

## Building
Building requires `make` and the go compiler. If you are running windows you can use [GnuMake32](https://gnuwin32.sourceforge.net/packages/make.htm).
Run for a release build:
```
make release
```
The resulting binary is `build/release/gocalsend`. The default build is in release mode, so you could also just run `make`.
For a development build with debug symbols and a bunch of odd test binaries:
```
make debug
```

## Related work
- [protocol](https://github.com/localsend/protocol) Protocol documentation
### Libraries
- [charmbracelet/log](https://github.com/charmbracelet/log) A slog compatible pretty logger
- [charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea) Awesome terminal ui framework
- [charmbracelet/bubbles](https://github.com/charmbracelet/bubbles) Useful stylable components
- [charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss) CSS like styling for bubbletea
- [go-toml](https://github.com/pelletier/go-toml/v2) toml parsing
### Alternative implementations
If your implementation is missing here, let me know
- [localsend](https://github.com/localsend/localsend) The reference implementation
- [localsend-rs](https://github.com/zpp0196/localsend-rs) Awesome rust tui 
- [go-localsend](https://github.com/meowrain/localsend-go) An alternative go implementation, check the docs folder for the english readme


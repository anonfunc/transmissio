[![Build Status](https://travis-ci.org/anonfunc/transmissio.svg?branch=master)](https://travis-ci.org/anonfunc/transmissio)
# Transmiss.io
A downloading service for Put.io

Project status:
- Perfectly useful blackhole downloader (supports .magnet files as well.)
- Somewhat useful Transmission RPC drop-in replacement
- Still very opaque: no visibility into downloading from Put.io besides
log messages, no configuration interface, etc.


## Setup from Source:
### Get an OAuth Token
- https://app.put.io/settings/account/oauth/apps/new
When done, you'll have a personal OAuth Token for the application.

### Docker: [anonfunc/transmissio](https://hub.docker.com/r/anonfunc/transmissio)
Mount /config, /download and /blackhole directories.

### Build:
- Install [go](https://golang.org/)
- Install [mage](https://magefile.org/)
- `mage build`
- `./transmissio`


### Config file
Create a config.yaml file in /config 
(or in the working directory, with out Docker):

    blackhole: /blackhole
    downloadTo: /download
    host: "0.0.0.0"
    port: "9091"
    oauth_token: OAUTH_TOKEN
    
### Run
If config was not found, a template config.yaml file is created.
  

## Using via blackhole directory

Place .magnet or .torrent files in the blackhole directory.
Subdirectories will be preserved in the download directory.

## Using via Transmission-RPC compatible API
Work in progress, but coming along.  Tested with nzb360.

Use `http://<address>:<port>` as the Transmission host, 
`/transmission/rpc` as the path if needed.  No auth,
so don't put this facing the internet.

Torrent status will reflect Put.io status, so a completed transfer
which is in the middle of downloading will appear to be 100% complete.
When finished downloading locally, the transfer will be removed after 10 minutes, 
not marked as seeding.   This is to support clients which need to be aware of the transfer
in order to do post-processing.

Handled RPC methods:

- session-get
- torrent-get
- torrent-add
- empty string (used as ping?)

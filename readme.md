# Transmiss.io (WORK IN PROGRESS)
A downloading service for Put.io

## Setup from Source:
### Get an OAuth Token
- https://app.put.io/settings/account/oauth/apps/new
When done, you'll have a personal OAuth Token for the application.

### Docker (not published yet):
Mount /config, /download and /blackhole directories.

### Or Build:
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
NOT READY YET.

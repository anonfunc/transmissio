package main

import (
	"log"
	"net/http"

	"github.com/anonfunc/transmissio/internal/pkg/torrent"

	"github.com/anonfunc/transmissio/internal/pkg/blackhole"
	"github.com/anonfunc/transmissio/internal/pkg/config"
	"github.com/spf13/viper"

	"github.com/anonfunc/transmissio/internal/pkg/transmission"
)

func main() {
	config.Config()
	transmission.Initialize()
	downloader := torrent.NewDownloader()
	transmission.Downloader = downloader
	go func() {
		blackhole.StartWatcher(downloader, viper.GetString("blackhole"))
	}()
	http.HandleFunc("/transmission/rpc", transmission.RPCHandler)
	listeningOn := viper.GetString("host") + ":" + viper.GetString("port")
	log.Printf("Listening on %s...", listeningOn)
	log.Fatal(http.ListenAndServe(listeningOn, nil))
}

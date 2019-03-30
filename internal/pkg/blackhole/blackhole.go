package blackhole

import (
	"log"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/anonfunc/transmissio/internal/pkg/torrent"

	"github.com/radovskyb/watcher"
)

func StartWatcher(downloader *torrent.PutIoDownloader, path string) {
	w := watcher.New()
	w.SetMaxEvents(1)
	r := regexp.MustCompile(`\.(torrent|magnet)$`)
	w.AddFilterHook(watcher.RegexFilterHook(r, true))

	go func() {
		for {
			select {
			case event := <-w.Event:
				handle(downloader, event, path) // Print the event's info.
			case err := <-w.Error:
				log.Fatalln(err)
			case <-w.Closed:
				return
			}
		}
	}()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Fatalf("blackhole directory %s does not exist", path)
	}
	// Watch this folder for changes.
	log.Printf("Watching %s for .torrent and .magnet files...", path)
	if err := w.AddRecursive(path); err != nil {
		log.Fatalln(err)
	}

	// Start the watching process - it'll check for changes every 100ms.
	if err := w.Start(time.Millisecond * 100); err != nil {
		log.Fatalln(err)
	}
}

func handle(downloader *torrent.PutIoDownloader, event watcher.Event, basePath string) {
	if event.Op != watcher.Create && event.Op != watcher.Write {
		return
	}
	downloadDir := blackholePathToDownloadDir(event.Path, basePath, viper.GetString("downloadTo"))
	ext := path.Ext(event.Path)
	switch ext {
	case ".torrent":
		downloader.AsyncFetchTorrent(event.Path, downloadDir)
	case ".magnet":
		downloader.AsyncFetchMagnetFile(event.Path, downloadDir)
	}
}

func blackholePathToDownloadDir(file string, basePath, baseDownloadDir string) string {
	return strings.Replace(path.Dir(file), strings.TrimRight(basePath, "/"), strings.TrimRight(baseDownloadDir, "/"), -1)
}

package torrent

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/anacrolix/torrent/metainfo"
	"github.com/igungor/go-putio/putio"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

type FetchResult struct {
	Error       error
	Name        string
	DownloadDir string
}

type PutIoDownloader struct {
	Client       *putio.Client
	PendingLinks chan string
	Results      chan FetchResult
}

func (r PutIoDownloader) AsyncFetchMagnetLink(urlStr string, downloadDir string) {
	go func() {
		result, _ := r.FetchMagnetLink(urlStr, downloadDir)
		r.Results <- result
	}()
}

func (r PutIoDownloader) AsyncFetchMagnetFile(filename, downloadDir string) {
	go func() {
		result, _ := r.FetchMagnetFile(filename, downloadDir)
		r.Results <- result
	}()
}

func (r PutIoDownloader) AsyncFetchTorrent(filename, downloadDir string) {
	go func() {
		result, _ := r.FetchTorrent(filename, downloadDir)
		r.Results <- result
	}()
}

func NewDownloader() *PutIoDownloader {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: viper.GetString("oauth_token")})
	oauthClient := oauth2.NewClient(context.Background(), tokenSource)
	downloader := &PutIoDownloader{
		Client:  putio.NewClient(oauthClient),
		Results: make(chan FetchResult, 100),
	}
	go func() {
		for {
			result := <-downloader.Results
			if result.Error == nil {
				log.Printf("Success: Downloaded %s to %s", result.Name, result.DownloadDir)
			} else {
				log.Printf("Failure: %v while downloading %s to %s", result.Error, result.Name, result.DownloadDir)
			}
		}
	}()
	return downloader
}

func (r PutIoDownloader) FetchMagnetFile(filename, downloadDir string) (FetchResult, error) {
	magnetLinkBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return FetchResult{Error: err}, err
	}
	result, err := r.FetchMagnetLink(string(magnetLinkBytes), downloadDir)
	renameOriginal(err, filename)
	return result, err
}

func renameOriginal(err error, filename string) {
	if err != nil {
		if err := os.Rename(filename, filename+".done"); err != nil {
			log.Printf("Unable to rename %s", filename)
		}
	} else {
		if err := os.Rename(filename, filename+".error"); err != nil {
			log.Printf("Unable to rename %s", filename)
		}
	}
}

func (r PutIoDownloader) FetchTorrent(filename, downloadDir string) (FetchResult, error) {
	magnetLink := torrentFileToMagnetLink(filename)
	if magnetLink == "" {
		err := fmt.Errorf("unable to fetch from torrent file %s", filename)
		return FetchResult{Error: err}, err
	}
	result, err := r.FetchMagnetLink(magnetLink, downloadDir)
	renameOriginal(err, filename)
	return result, err
}

func (r PutIoDownloader) FetchMagnetLink(urlStr string, downloadDir string) (FetchResult, error) {
	transfer, err := r.Client.Transfers.Add(context.TODO(), urlStr, -1, "")
	if err != nil {
		return FetchResult{Error: err}, err
	}
	startTime := time.Now()
	for {
		if time.Now().After(startTime.Add(24 * time.Hour)) {
			// After 24 hours, bail.
			err := fmt.Errorf("transfer for %s taking too long, cancelling", transfer.Name)
			return FetchResult{Error: err}, err
		}
		updated, err := r.Client.Transfers.Get(context.TODO(), transfer.ID)
		// fmt.Printf("%v\n", updated)
		if err != nil {
			return FetchResult{Error: err}, err
		}
		if updated.Status == "COMPLETED" || updated.Status == "SEEDING" {
			if err := r.downloadCompletedTorrent(updated, downloadDir); err != nil {
				return FetchResult{Error: err}, err
			}
			if err := r.Client.Files.Delete(context.TODO(), updated.FileID); err != nil {
				log.Printf("Unable to remove completed download! %s", updated.Name)
			}
			if err := r.Client.Transfers.Clean(context.TODO()); err != nil {
				log.Printf("Unable to clean transfer list! %s", updated.Name)
			}
			return FetchResult{Error: err, Name: transfer.Name, DownloadDir: downloadDir}, nil
		}
		log.Printf("Sleeping %d seconds for %s ...", 10+updated.EstimatedTime/4, transfer.Name)
		time.Sleep(time.Duration(10+updated.EstimatedTime/4) * time.Second)
	}
}

func (r PutIoDownloader) downloadCompletedTorrent(updated putio.Transfer, downloadDir string) error {
	log.Printf("Starting download of %s to %s", updated.Name, downloadDir)
	file, err := r.Client.Files.Get(context.TODO(), updated.FileID)
	if err != nil {
		return err
	}
	if err := r.recursiveDownload(file, downloadDir); err != nil {
		return err
	}
	return nil
}

func (r PutIoDownloader) recursiveDownload(file putio.File, downloadDir string) error {
	if file.ContentType == "application/x-directory" {
		children, _, err := r.Client.Files.List(context.TODO(), file.ID)
		if err != nil {
			return err
		}
		for _, child := range children {
			if err := r.recursiveDownload(child, filepath.Join(downloadDir, file.Name)); err != nil {
				return err
			}
		}
	} else {
		if err := r.downloadFile(file, downloadDir); err != nil {
			return err
		}
	}
	return nil
}

func (r PutIoDownloader) downloadFile(file putio.File, downloadDir string) error {
	readCloser, err := r.Client.Files.Download(context.TODO(), file.ID, true, nil)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(downloadDir, 0777); err != nil {
		return err
	}
	downloadFilename := filepath.Join(downloadDir, file.Name)
	outFile, err := os.Create(downloadFilename)
	if err != nil {
		return err
	}
	defer outFile.Close()
	_, err = io.Copy(outFile, readCloser)
	if err != nil {
		return err
	}
	log.Printf("Done with download of %s to %s", file.Name, downloadDir)
	return nil
}

func torrentFileToMagnetLink(filename string) string {
	mi, err := metainfo.LoadFromFile(filename)
	if err != nil {
		log.Printf("error loading torrent file: %s", err.Error())
		return ""
	}
	info, err := mi.UnmarshalInfo()
	if err != nil {
		log.Printf("error converting torrent: %s", err.Error())
		return ""
	}

	return mi.Magnet(info.Name, mi.HashInfoBytes()).String()
}

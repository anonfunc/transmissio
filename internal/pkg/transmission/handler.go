// https://github.com/transmission/transmission/blob/2.9x/extras/rpc-spec.txt
package transmission

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/anacrolix/torrent/metainfo"
	"github.com/anonfunc/transmissio/internal/pkg/torrent"
	"github.com/spf13/viper"

	"golang.org/x/sys/unix"
)

const sessionIDHeader = "X-Transmission-Session-Id"

var knownSessionID string
var Downloader *torrent.PutIoDownloader

func Initialize() {
	sessionRandomBytes := make([]byte, 16)
	_, err := rand.Read(sessionRandomBytes)
	if err != nil {
		panic(err)
	}
	knownSessionID = base64.StdEncoding.EncodeToString(sessionRandomBytes)
	unix.Umask(000) // TODO Is this appropriate outside of Docker?
}

func (receiver *RPCRequest) DoIt() (*RPCResponse, error) {
	response := &RPCResponse{Tag: receiver.Tag}
	switch receiver.Method {
	case "session-get":
		response.Arguments = SessionInfo{
			Version:           "2.98",
			RPCVersion:        "10",
			RPCVersionMinimum: "10",
			SpeedLimitDown:    10000,
			SpeedLimitUp:      10000,
		}
	// https://github.com/transmission/transmission/blob/2.9x/extras/rpc-spec.txt#L86
	case "torrent-start":
	case "torrent-start-now":
	case "torrent-stop":
	case "torrent-verify":
	case "torrent-reannounce":
	case "torrent-set":
		// https://github.com/transmission/transmission/blob/2.9x/extras/rpc-spec.txt#L105
	case "torrent-get":
		// https://github.com/transmission/transmission/blob/2.9x/extras/rpc-spec.txt#L144
		response.Arguments = receiver.torrentGet()
	case "torrent-add":
		response.Arguments = receiver.torrentAdd()
	case "torrent-remove":
		// https://github.com/transmission/transmission/blob/2.9x/extras/rpc-spec.txt#L407
	case "torrent-set-location":
		// https://github.com/transmission/transmission/blob/2.9x/extras/rpc-spec.txt#L423
	case "torrent-rename-path":
		// https://github.com/transmission/transmission/blob/2.9x/extras/rpc-spec.txt#L440
	// Session stuff, make no-ops?
	case "free-space":
		// https://github.com/transmission/transmission/blob/2.9x/extras/rpc-spec.txt#L623
		// Should this be put.io space?
	case "":
		// Ping from nzb360 et al.
	default:
		log.Printf("unhandled method %s", receiver.Method)
	}
	response.Result = "success"
	return response, nil
}

func (receiver *RPCRequest) torrentAdd() (result TorrentAdd) {
	filename := receiver.Arguments["filename"].(string)
	metainfoI := receiver.Arguments["metainfo"]
	downloadTo := viper.GetString("downloadTo")
	if strings.HasPrefix(filename, "magnet:") {
		Downloader.AsyncFetchMagnetLink(filename, downloadTo)
	} else if metainfoI != nil {
		metaBytes, err := base64.StdEncoding.DecodeString(metainfoI.(string))
		if err != nil {
			return
		}
		mi, err := metainfo.Load(bytes.NewBuffer(metaBytes))
		if err != nil {
			log.Printf("error loading torrent base64: %s", err.Error())
			return
		}
		info, err := mi.UnmarshalInfo()
		if err != nil {
			log.Printf("error converting torrent: %s", err.Error())
			return
		}
		magnetLink := mi.Magnet(info.Name, mi.HashInfoBytes()).String()
		Downloader.AsyncFetchMagnetLink(magnetLink, downloadTo)
	}
	return result
}

func (receiver *RPCRequest) torrentGet() TorrentGet {
	result := TorrentGet{
		Torrents: []TorrentInfo{},
	}
	transfers, err := Downloader.Client.Transfers.List(context.TODO())
	if err != nil {
		log.Printf("error in torrentGet: %s", err.Error())
		return result
	}
	for _, transfer := range transfers {
		var status int64
		switch transfer.Status {
		case "DOWNLOADING":
			status = 4
		case "IN_QUEUE":
			status = 3
		case "COMPLETED":
			status = 0
		case "SEEDING":
			status = 6
		default:
			log.Printf("unknown status %s", transfer.Status)
			status = 1
		}
		torrentInfo := TorrentInfo{
			ID:                 transfer.ID,
			Name:               transfer.Name,
			Error:              0,
			ErrorString:        transfer.ErrorMessage,
			Status:             status,
			DownloadDir:        viper.GetString("downloadTo"),
			RateDownload:       int64(transfer.DownloadSpeed),
			RateUpload:         int64(transfer.UploadSpeed),
			PeersGettingFromUs: int64(transfer.PeersGettingFromUs),
			PeersSendingToUs:   int64(transfer.PeersSendingToUs),
			PeersConnected:     int64(transfer.PeersConnected),
			Eta:                transfer.EstimatedTime,
			HaveUnchecked:      0,
			HaveValid:          transfer.Downloaded,
			UploadedEver:       transfer.Uploaded,
			SizeWhenDone:       int64(transfer.Size),
			DesiredAvailable:   int64(transfer.Availability),
			Comment:            transfer.StatusMessage,
		}
		if transfer.CreatedAt != nil {
			torrentInfo.AddedDate = transfer.CreatedAt.Unix()
		}
		if transfer.FinishedAt != nil {
			torrentInfo.DoneDate = transfer.FinishedAt.Unix()
		}
		result.Torrents = append(result.Torrents, torrentInfo)
	}
	return result
}

func RPCHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	sessionID := r.Header.Get(sessionIDHeader)
	if len(sessionID) == 0 {
		w.Header().Set(sessionIDHeader, knownSessionID)
		// w.WriteHeader(http.StatusConflict)
		// return
	}

	bytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("hit! %s\n", string(bytes))
	var rpcRequest RPCRequest
	if err := json.Unmarshal(bytes, &rpcRequest); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	response, err := rpcRequest.DoIt()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	responseBytes, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Response: %s", string(responseBytes))
	if _, err := w.Write(responseBytes); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

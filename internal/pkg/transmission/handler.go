// https://github.com/transmission/transmission/blob/2.9x/extras/rpc-spec.txt
package transmission

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"hash/fnv"
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
	var downloadTo string
	switch v := receiver.Arguments["download-dir"].(type) {
	case string:
		downloadTo = v
	default:
		downloadTo = viper.GetString("downloadTo")
	}
	if strings.HasPrefix(filename, "magnet:") {
		Downloader.AsyncFetchMagnetLink(filename, downloadTo)
		mi, err := metainfo.ParseMagnetURI(filename)
		if err != nil {
			log.Printf("Unable to parse magnet URI")
			return
		}
		h := fnv.New32a()
		if _, err := h.Write(mi.InfoHash.Bytes()); err != nil {
			log.Printf("Unable to make filename into ID, %s\n", err.Error())
		}
		id := int64(h.Sum32())
		result.TorrentAdded = &TorrentInfoSmall{
			ID:         id,
			Name:       mi.DisplayName,
			HashString: mi.InfoHash.HexString(),
		}
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
		hashBytes := mi.HashInfoBytes()
		h := fnv.New32a()
		if _, err := h.Write(hashBytes.Bytes()); err != nil {
			log.Printf("Unable to make hash into ID, %s\n", err.Error())
		}
		result.TorrentAdded = &TorrentInfoSmall{
			ID:         int64(h.Sum32()),
			Name:       filename,
			HashString: hashBytes.HexString(),
		}
		magnetLink := mi.Magnet(info.Name, hashBytes).String()
		Downloader.AsyncFetchMagnetLink(magnetLink, downloadTo)
	}
	return result
}

func (receiver *RPCRequest) torrentGet() TorrentGet {
	transfers, err := Downloader.Client.Transfers.List(context.TODO())
	if err != nil {
		log.Printf("error in torrentGet: %s", err.Error())
		return TorrentGet{}
	}
	idsSearchRaw, idsSearchGiven := receiver.Arguments["ids"]
	idsSearch, idSearchIsSlice := idsSearchRaw.([]interface{})
	torrents := make([]TorrentInfo, 0, len(transfers))
	for _, transfer := range transfers {
		var status int64
		switch transfer.Status {
		case "DOWNLOADING":
			status = 4
		case "IN_QUEUE":
			status = 3
		case "COMPLETED":
			status = 4 // Downloading makes sense, since we clear after we DL.
			transfer.Downloaded = int64(transfer.Size)
			transfer.Availability = 100
		case "SEEDING":
			status = 6
		default:
			log.Printf("unknown status %s", transfer.Status)
			status = 7
		}
		var id int64
		var hash string
		if transfer.MagnetURI != "" {
			mi, err := metainfo.ParseMagnetURI(transfer.MagnetURI)
			if err != nil {
				log.Printf("Unabled to parse magnet URI %s", err.Error())
				id = transfer.ID
				hash = "hash"
			} else {
				h := fnv.New32a()
				if _, err := h.Write(mi.InfoHash.Bytes()); err != nil {
					log.Printf("Unable to make filename into ID, %s\n", err.Error())
				}
				id = int64(h.Sum32())
				hash = mi.InfoHash.HexString()
			}
		} else {
			// TODO Stable ID and hash for torrent files.
			log.Printf("No magnetURI for transfer.")
			id = transfer.ID
			hash = "hash"
		}

		if idsSearchGiven && idSearchIsSlice {
			var match bool
		outer:
			for _, idRaw := range idsSearch {
				switch searchID := idRaw.(type) {
				case int64:
					if id == searchID {
						match = true
						break outer
					}
				case string:
					if hash == searchID {
						match = true
						break outer
					}
				default:
					// TODO
					match = true
				}
			}
			if !match {
				continue
			}
		}

		torrentInfo := TorrentInfo{
			ID:                 id,
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
		torrents = append(torrents, torrentInfo)
	}
	return TorrentGet{
		Torrents: torrents,
	}
}

func RPCHandler(w http.ResponseWriter, r *http.Request) {
	requestBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Request: %s", string(requestBytes))
	sessionID := r.Header.Get(sessionIDHeader)
	if len(sessionID) == 0 {
		w.Header().Set(sessionIDHeader, knownSessionID)
		w.WriteHeader(http.StatusConflict)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var rpcRequest RPCRequest
	if err := json.Unmarshal(requestBytes, &rpcRequest); err != nil {
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

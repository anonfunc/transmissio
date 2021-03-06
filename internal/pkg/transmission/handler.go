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
			HashString: strings.ToLower(mi.InfoHash.HexString()),
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
			HashString: strings.ToLower(hashBytes.HexString()),
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
	var fields []interface{}
	if receiver.Arguments["fields"] != nil {
		fields = receiver.Arguments["fields"].([]interface{})
	}
	idsSearchRaw, idsSearchGiven := receiver.Arguments["ids"]
	idsSearch, idSearchIsSlice := idsSearchRaw.([]interface{})
	torrents := make([]TorrentInfo, 0, len(transfers))
	for _, transfer := range transfers {
		log.Printf("Active Transfer: %v", transfer)
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
		if strings.HasPrefix(transfer.Source, "magnet:") {
			mi, err := metainfo.ParseMagnetURI(transfer.Source)
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
				hash = strings.ToLower(mi.InfoHash.HexString())
			}
		} else {
			// TODO Stable ID and hash for torrent files.
			id, hash = torrentLinkToIDAndHash(transfer.Source)
			log.Printf("No magnet URI, fetched and derived %d and %s", id, hash)
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
					if hash == strings.ToLower(searchID) {
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

		torrentInfo := TorrentInfo{}
		for _, vi := range fields {
			v := vi.(string)
			switch {
			case v == "id":
				torrentInfo.ID = id
			case v == "name":
				torrentInfo.Name = transfer.Name
			case v == "error":
				var i int64 // TODO What to put here?
				torrentInfo.Error = &i
			case v == "errorString":
				torrentInfo.ErrorString = transfer.StatusMessage
			case v == "status":
				torrentInfo.Status = status
			case v == "downloadDir":
				torrentInfo.DownloadDir = viper.GetString("downloadTo")
			case v == "rateDownload":
				i := int64(transfer.DownloadSpeed)
				torrentInfo.RateDownload = &i
			case v == "rateUpload":
				i := int64(transfer.UploadSpeed)
				torrentInfo.RateUpload = &i
			case v == "peersGettingFromUs":
				i := int64(transfer.PeersGettingFromUs)
				torrentInfo.PeersGettingFromUs = &i
			case v == "peersSendingToUs":
				i := int64(transfer.PeersSendingToUs)
				torrentInfo.PeersSendingToUs = &i
			case v == "peersConnected":
				i := int64(transfer.PeersConnected)
				torrentInfo.PeersConnected = &i
			case v == "eta":
				torrentInfo.Eta = transfer.EstimatedTime
			case v == "haveValid":
				torrentInfo.HaveValid = &transfer.Downloaded
			case v == "uploadedEver":
				torrentInfo.UploadedEver = &transfer.Uploaded
			case v == "sizeWhenDone":
				torrentInfo.SizeWhenDone = int64(transfer.Size)
			case v == "desiredAvailable":
				i := int64(transfer.Availability)
				torrentInfo.DesiredAvailable = &i
			case v == "comment":
				torrentInfo.Comment = transfer.StatusMessage
			case v == "percentDone":
				torrentInfo.PercentDone = float32(transfer.Downloaded) / float32(transfer.Size)
			case v == "isFinished":
				torrentInfo.IsFinished = status >= 4
			case v == "addedDate":
				if transfer.CreatedAt != nil {
					torrentInfo.AddedDate = transfer.CreatedAt.Unix()
				}
			case v == "doneDate":
				if transfer.FinishedAt != nil {
					torrentInfo.DoneDate = transfer.FinishedAt.Unix()
				}
			case v == "files":
				files, err := Downloader.RecursiveList(transfer.FileID, viper.GetString("downloadTo"))
				if err != nil {
					log.Printf("error listing files, %s", err.Error())
					continue
				}
				for _, f := range files {
					torrentInfo.Files = append(torrentInfo.Files, FileInfo{
						BytesCompleted: f.Size * transfer.Downloaded / int64(transfer.Size), // Fake percentage.
						Length:         f.Size,
						Name:           f.Name,
					})
				}
			}
		}
		log.Printf("ti: %v", torrentInfo)
		torrents = append(torrents, torrentInfo)
	}
	return TorrentGet{
		Torrents: torrents,
	}
}

var torrentLinkInfoCache = make(map[string]*metainfo.MetaInfo)

func torrentLinkToIDAndHash(torrentLink string) (int64, string) {
	if torrentLinkInfoCache[torrentLink] == nil {
		log.Printf("Retrieving torrent %s", torrentLink)
		// TODO Smarter max cache size.
		if len(torrentLinkInfoCache) > 1000 {
			torrentLinkInfoCache = make(map[string]*metainfo.MetaInfo)
		}
		resp, err := http.Get(torrentLink) //nolint:gosec
		if err != nil {
			log.Printf("Error retrieving torrent link: %s", err.Error())
			return 0, ""
		}
		body, err := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			log.Printf("Error reading torrent response: %s", err.Error())
			return 0, ""
		}
		info, err := metainfo.Load(bytes.NewBuffer(body))
		if err != nil {
			log.Printf("Error parsing torrent response: %s", err.Error())
			log.Printf("Torrent response: %s", string(body))
			return 0, ""
		}
		torrentLinkInfoCache[torrentLink] = info
	}
	hashInfo := torrentLinkInfoCache[torrentLink].HashInfoBytes()
	h := fnv.New32a()
	if _, err := h.Write(hashInfo.Bytes()); err != nil {
		log.Printf("Unable to make filename into ID, %s\n", err.Error())
	}
	id := int64(h.Sum32())
	hash := strings.ToLower(hashInfo.HexString())
	return id, hash

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

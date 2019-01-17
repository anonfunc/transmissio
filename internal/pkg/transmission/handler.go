// https://github.com/transmission/transmission/blob/2.9x/extras/rpc-spec.txt
package transmission

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

const sessionIDHeader = "X-Transmission-Session-Id"

var knownSessionID string

func Initialize() {
	sessionRandomBytes := make([]byte, 16)
	_, err := rand.Read(sessionRandomBytes)
	if err != nil {
		panic(err)
	}
	knownSessionID = base64.StdEncoding.EncodeToString(sessionRandomBytes)
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
		response.Arguments = TorrentGet{}
	case "torrent-add":
		// https://github.com/transmission/transmission/blob/2.9x/extras/rpc-spec.txt#L371
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
	default:
		log.Printf("unhandled method %s", receiver.Method)
	}
	response.Result = "success"
	return response, nil
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
	if _, err := w.Write(responseBytes); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

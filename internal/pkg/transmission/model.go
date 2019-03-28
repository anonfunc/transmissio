package transmission

type RPCRequest struct {
	Method    string                 `json:"method"`
	Arguments map[string]interface{} `json:"arguments"`
	Tag       *int                   `json:"tag,omitempty"`
}

type RPCResponse struct {
	Result    string      `json:"result"`
	Arguments interface{} `json:"arguments,omitempty"`
	Tag       *int        `json:"tag,omitempty"`
}

type SessionInfo struct {
	Version               string `json:"version"`
	RPCVersion            string `json:"rpc-version"`
	RPCVersionMinimum     string `json:"rpc-version-minimum"`
	SpeedLimitDown        int    `json:"speed-limit-down"`
	SpeedLimitUp          int    `json:"speed-limit-up"`
	SpeedLimitDownEnabled bool   `json:"speed-limit-down-enabled"`
	SpeedLimitUpEnabled   bool   `json:"speed-limit-up-enabled"`
}

type TorrentGet struct {
	Torrents []TorrentInfo `json:"torrents"`
	Removed  []int         `json:"removed,omitempty"`
}

type TorrentInfo struct {
	ID                 int64  `json:"id,omitempty"`
	Name               string `json:"name,omitempty"`
	Error              *int64 `json:"error,omitempty"`
	ErrorString        string `json:"errorString,omitempty"`
	Status             int64  `json:"status,omitempty"` // tr_torrent_activity
	DownloadDir        string `json:"downloadDir,omitempty"`
	RateDownload       *int64 `json:"rateDownload,omitempty"` // (B/s)
	RateUpload         *int64 `json:"rateUpload,omitempty"`   // (B/s)
	PeersGettingFromUs *int64 `json:"peersGettingFromUs,omitempty"`
	PeersSendingToUs   *int64 `json:"peersSendingToUs,omitempty"`
	PeersConnected     *int64 `json:"peersConnected,omitempty"`
	Eta                int64  `json:"eta,omitempty"`
	// HaveUnchecked      *int64      `json:"haveUnchecked,omitempty"`
	HaveValid        *int64     `json:"haveValid,omitempty"`
	UploadedEver     *int64     `json:"uploadedEver,omitempty"`
	SizeWhenDone     int64      `json:"sizeWhenDone,omitempty"`
	AddedDate        int64      `json:"addedDate,omitempty"`
	DoneDate         int64      `json:"doneDate,omitempty"`
	DesiredAvailable *int64     `json:"desiredAvailable,omitempty"`
	Comment          string     `json:"comment,omitempty"`
	PercentDone      float32    `json:"percentDone,omitempty"`
	IsFinished       bool       `json:"isFinished,omitempty"`
	Files            []FileInfo `json:"files,omitempty"`
}

type FileInfo struct {
	BytesCompleted int64  `json:"bytesCompleted"`
	Length         int64  `json:"length"`
	Name           string `json:"name"`
}

type TorrentInfoSmall struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	HashString string `json:"hashString"`
}

type TorrentAdd struct {
	TorrentAdded     *TorrentInfoSmall `json:"torrent-added,omitempty"`
	TorrentDuplicate *TorrentInfoSmall `json:"torrent-duplicate,omitempty"`
}

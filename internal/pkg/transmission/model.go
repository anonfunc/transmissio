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
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	Error              int64  `json:"error"`
	ErrorString        string `json:"errorString"`
	Status             int64  `json:"status"` // tr_torrent_activity
	DownloadDir        string `json:"downloadDir"`
	RateDownload       int64  `json:"rateDownload"` // (B/s)
	RateUpload         int64  `json:"rateUpload"`   // (B/s)
	PeersGettingFromUs int64  `json:"peersGettingFromUs"`
	PeersSendingToUs   int64  `json:"peersSendingToUs"`
	PeersConnected     int64  `json:"peersConnected"`
	Eta                int64  `json:"eta"`
	HaveUnchecked      int64  `json:"haveUnchecked"`
	HaveValid          int64  `json:"haveValid"`
	UploadedEver       int64  `json:"uploadedEver"`
	SizeWhenDone       int64  `json:"sizeWhenDone"`
	AddedDate          int64  `json:"addedDate"`
	DoneDate           int64  `json:"doneDate"`
	DesiredAvailable   int64  `json:"desiredAvailable"`
	Comment            string `json:"comment"`
	PercentDone        float32  `json:"percentDone"`
	IsFinished         bool   `json:"isFinished"`
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

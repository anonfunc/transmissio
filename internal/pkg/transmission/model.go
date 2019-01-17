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
	Torrents []TorrentInfo `json:"torrents,omitempty"`
	Removed  []int         `json:"removed,omitempty"`
}

type TorrentInfo struct {
}

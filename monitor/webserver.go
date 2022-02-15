package monitor

import (
	"encoding/json"
	"net/http"
	_ "net/http/pprof"
	"time"
)

type MonitorWebserver struct {
	Monitor     *ReorgMonitor
	Addr        string
	TimeStarted time.Time
}

// API response
type StatusResponse struct {
	Monitor     MonitorInfo
	Connections []ConnectionInfo
}

type MonitorInfo struct {
	Id                  string
	NumBlocks           int
	EarliestBlockNumber uint64
	LatestBlockNumber   uint64
	TimeStarted         string
}

type ConnectionInfo struct {
	NodeUri         string
	IsConnected     bool
	IsSubscribed    bool
	NumBlocks       uint64
	NumReconnects   int64
	NumResubscribes int64
	NextTimeout     int64
}

func NewMonitorWebserver(monitor *ReorgMonitor, listenAddr string) *MonitorWebserver {
	return &MonitorWebserver{
		Monitor:     monitor,
		Addr:        listenAddr,
		TimeStarted: time.Now().UTC(),
	}
}

func (ws *MonitorWebserver) HandleStatusRequest(w http.ResponseWriter, r *http.Request) {
	// fmt.Fprintf(w, "Monitor: %s\n", ws.Monitor.String())
	res := StatusResponse{
		Monitor: MonitorInfo{
			Id:                  ws.Monitor.String(),
			NumBlocks:           len(ws.Monitor.BlockByHash),
			EarliestBlockNumber: ws.Monitor.EarliestBlockNumber,
			LatestBlockNumber:   ws.Monitor.LatestBlockNumber,
			TimeStarted:         ws.TimeStarted.String(),
		},
		Connections: make([]ConnectionInfo, 0),
	}

	for _, c := range ws.Monitor.connections {
		connInfo := ConnectionInfo{
			NodeUri:         c.NodeUri,
			IsConnected:     c.IsConnected,
			IsSubscribed:    c.IsSubscribed,
			NumBlocks:       c.NumBlocks,
			NumReconnects:   c.NumReconnects,
			NumResubscribes: c.NumResubscribes,
			NextTimeout:     c.NextRetryTimeoutSec,
		}
		res.Connections = append(res.Connections, connInfo)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)

}

func (ws *MonitorWebserver) ListenAndServe() error {
	http.HandleFunc("/", ws.HandleStatusRequest)
	return http.ListenAndServe(ws.Addr, nil)
}

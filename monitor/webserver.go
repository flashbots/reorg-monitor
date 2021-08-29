package monitor

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type MonitorWebserver struct {
	Monitor *ReorgMonitor
	Addr    string
}

type MonitorInfo struct {
	Id                  string
	NumBlocks           int
	EarliestBlockNumber uint64
	LatestBlockNumber   uint64
}

type ConnectionInfo struct {
	NodeUri         string
	IsConnected     bool
	IsSubscribed    bool
	NumBlocks       uint64
	NumReconnects   int64
	NumResubscribes int64
}

type StatusResponse struct {
	Monitor     MonitorInfo
	Connections []ConnectionInfo
}

func NewMonitorWebserver(monitor *ReorgMonitor, port int) *MonitorWebserver {
	return &MonitorWebserver{
		Monitor: monitor,
		Addr:    fmt.Sprintf(":%d", port),
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

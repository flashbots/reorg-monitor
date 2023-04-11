package monitor

import (
	"context"
	"github.com/r3labs/sse/v2"
	"log"
	"net/url"
)

const (
	events          = "/eth/v1/events"
	chainReorgEvent = "chain_reorg"
)

type Mon struct {
	clClients []*sse.Client // consensus layer clients
}

func NewMonitor(elURIs, clURIs []string) (*Mon, error) {
	clClients := make([]*sse.Client, 0)

	for _, uri := range clURIs {
		u, err := url.JoinPath(uri, events)
		if err != nil {
			return nil, err
		}

		client := sse.NewClient(u)
		clClients = append(clClients, client)
	}

	return &Mon{clClients: clClients}, nil
}

func (m *Mon) ListenAndServe(ctx context.Context) error {
	for _, cl := range m.clClients {
		err := cl.Subscribe(chainReorgEvent, func(msg *sse.Event) {
			log.Printf("id: %s, data: %s", msg.ID, msg.Data)
		})
		if err != nil {
			return err
		}
	}

	return nil
}

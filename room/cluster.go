package room

import (
	"os"
	"os/signal"
	"time"
)

type commandPayload struct {
	action commandAction
	id     string
	result chan<- *Hub
	all    chan<- []string
	data   []byte
	hub    *Hub
}

type Cluster struct {
	isDying  bool
	General  *Hub
	listener chan commandPayload
	pool     map[string]*Hub
}

func NewCluster() *Cluster {
	cluster := Cluster{
		listener: make(chan commandPayload),
		pool:     make(map[string]*Hub),
		General:  nil,
	}
	cluster.General = NewHub("general", &cluster)
	go cluster.run()
	go func() {
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt)
		<-interrupt
		time.Sleep(time.Second * 2)
		os.Exit(0)
	}()
	return &cluster
}

func (cluster *Cluster) run() {
	for command := range cluster.listener {
		switch command.action {
		case add:
			cluster.pool[command.hub.ID] = command.hub
		case get:
			command.result <- cluster.pool[command.id]
		case remove:
			delete(cluster.pool, command.id)
		case emit:
			for _, hub := range cluster.pool {
				if hub.ID == command.id {
					hub.Emit(command.data)
					break
				}
			}
		case die:
			return
		case all:
			var result []string
			for _, hub := range cluster.pool {
				result = append(result, hub.ID)
			}
			command.all <- result
		}
	}
}

func (cluster *Cluster) All() []string {
	result := make(chan []string)
	cluster.listener <- commandPayload{
		action: all,
		all:    result,
	}
	return <-result
}

func (cluster *Cluster) Add(hub *Hub) {
	cluster.listener <- commandPayload{
		action: add,
		hub:    hub,
	}
}

func (cluster *Cluster) Get(id string) *Hub {
	result := make(chan *Hub)
	cluster.listener <- commandPayload{
		action: get,
		id:     id,
		result: result,
	}
	return <-result
}

func (cluster *Cluster) Remove(id string) {
	cluster.listener <- commandPayload{
		action: remove,
		id:     id,
	}
	consumeHubRemoved(cluster, EventHubRemoved{
		EventHead: &EventHead{
			Action: EVENT_HUB_REMOVED,
			Id:     randomId(IdLength),
			To:     TO_EVERYONE,
		},
		Payload: HubRemovedPayload{
			Name: id,
		},
	})
}

func (cluster *Cluster) Emit(msg []byte, hubID string) {
	cluster.listener <- commandPayload{
		action: emit,
		id:     hubID,
		data:   msg,
	}
}

func (cluster *Cluster) Die() {
	cluster.listener <- commandPayload{
		action: die,
	}
}

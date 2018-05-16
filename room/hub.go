package room

import (
	"fmt"
	"log"
)

const (
	remove commandAction = iota
	emit
	add
	get
	length
	die
	all
)

type commandAction int

type commandData struct {
	action  commandAction
	key     string
	storeId int
	result  chan<- *Client
	length  chan<- int
	all     chan<- []string
	data    []byte
	client  *Client
}

type Hub struct {
	listener chan commandData
	pool     map[string]*Client
	cluster  *Cluster
	ID       string
}

func NewHub(id string, cluster *Cluster) *Hub {
	hub := Hub{
		listener: make(chan commandData),
		pool:     make(map[string]*Client),
		ID:       id,
		cluster:  cluster,
	}
	log.Println(fmt.Sprintf("Hub with ID %s created...", hub.ID))
	go hub.run()
	return &hub
}

func (hub *Hub) run() {
	for command := range hub.listener {
		switch command.action {
		case all:
			var all []string
			for _, client := range hub.pool {
				all = append(all, client.Name)
			}
			command.all <- all
		case add:
			hub.pool[command.client.Name] = command.client
			command.client.attachToHub(hub)
		case get:
			command.result <- hub.pool[command.key]
		case remove:
			delete(hub.pool, command.key)
			data := getClientRemoved(command.key)
			for _, client := range hub.pool {
				client.Send(data)
			}
		case length:
			command.length <- len(hub.pool)
		case emit:
			for _, client := range hub.pool {
				client.Send(command.data)
			}
		case die:
			return
		}
	}
}

func (hub *Hub) All() []string {
	result := make(chan []string)
	hub.listener <- commandData{
		action: all,
		all:    result,
	}
	return <-result
}

func (hub *Hub) Add(client *Client) {
	hub.listener <- commandData{
		action: add,
		client: client,
	}
}

func (hub *Hub) Get(key string) *Client {
	result := make(chan *Client)
	hub.listener <- commandData{
		action: get,
		key:    key,
		result: result,
	}
	return <-result
}

func (hub *Hub) Length() int {
	lnth := make(chan int)
	hub.listener <- commandData{
		action: length,
		length: lnth,
	}
	return <-lnth
}

func (hub *Hub) Remove(key string) {
	hub.listener <- commandData{
		action: remove,
		key:    key,
	}
}

func (hub *Hub) Emit(msg []byte) {
	hub.listener <- commandData{
		action: emit,
		data:   msg,
	}
}

func (hub *Hub) Die() {
	log.Println(fmt.Sprintf("Hub with ID %s removed...", hub.ID))
	hub.listener <- commandData{
		action: die,
	}
}

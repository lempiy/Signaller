package room

import (
	"github.com/json-iterator/go"
	"log"
)

type Client struct {
	send        chan<- []byte
	read        <-chan []byte
	Name        string
	Key         string
	die         <-chan struct{}
	Hub         *Hub
	hubListener chan<- commandData
}

func NewClient(send chan<- []byte, read <-chan []byte, die <-chan struct{}, name string) *Client {
	c := &Client{
		send: send,
		read: read,
		Name: name,
		Key:  randomId(12),
		die:  die,
	}
	log.Println("Client " + c.Name + " connected...")
	go c.watch()
	return c
}

func (c *Client) attachToHub(hub *Hub) {
	if c.Hub != nil {
		c.hubListener <- commandData{
			action: remove,
			key:    c.Key,
		}
	}
	c.Hub = hub
	c.hubListener = hub.listener
}

func (c *Client) watch() {
	for {
		select {
		case <-c.die:
			c.Die()
			return
		case msg := <-c.read:
			var event Event
			log.Println("Client "+c.Name+" read: ", string(msg))
			if err := jsoniter.Unmarshal(msg, &event); err != nil {
				log.Println("Client "+c.Name+" read error: ", err)
			}
			ConsumeEvent(c, event)
		}
	}
}

func (c *Client) Send(msg []byte) {
	c.send <- msg
}

func (c *Client) Die() {
	log.Println("Client " + c.Name + " disconnected...")
	if c.hubListener != nil {
		c.hubListener <- commandData{
			action: remove,
			key:    c.Key,
		}
	}
}

package room

import "log"

type Client struct {
	send        chan<- []byte
	read        <-chan []byte
	name       string
	Key         string
	UUID        string
	die         <-chan struct{}
	hub         *Hub
	hubListener chan<- commandData
}

func NewClient(send chan<- []byte, read <-chan []byte, die <-chan struct{}, name string) *Client {
	c := &Client{
		send:    send,
		read:    read,
		name:   name,
		die:     die,
	}
	log.Println("Client " + c.Key + " connected...")
	go c.watch()
	return c
}

func (c *Client) attachToHub(hub *Hub) {
	if c.hub != nil {
		c.hubListener <- commandData{
			action: remove,
			key:    c.Key,
		}
	}
	c.hub = hub
	c.hubListener = hub.listener
}

func (c *Client) watch() {
	for {
		select {
		case <-c.die:
			c.Die()
			return
		case msg := <-c.read:
			// Do stuff
			log.Println("Client "+c.Key+" read: ", string(msg))
		}
	}
}

func (c *Client) Send(msg []byte) {
	c.send <- msg
}

func (c *Client) Die() {
	log.Println("Client " + c.Key + " disconnected...")
	if c.hubListener != nil {
		c.hubListener <- commandData{
			action: remove,
			key:    c.Key,
		}
	}
}

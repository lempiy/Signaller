package room

import (
	"fmt"
	"github.com/json-iterator/go"
	"log"
	"math/rand"
	"time"
)

const (
	TO_EVERYONE = "*"

	EVENT_NEW_HUB_REQUEST = "EVENT_NEW_HUB_REQUEST"
	EVENT_NEW_HUB_CREATED = "EVENT_NEW_HUB_CREATED"
	EVENT_HUB_REMOVED     = "EVENT_HUB_REMOVED"
	EVENT_GET_HUBS        = "EVENT_GET_HUBS"

	EVENT_OFFER_CONNECTION     = "EVENT_OFFER_CONNECTION"
	EVENT_ANSWER_CONNECTION    = "EVENT_ANSWER_CONNECTION"
	EVENT_CANDIDATE_CONNECTION = "EVENT_CANDIDATE_CONNECTION"

	EVENT_ERROR = "EVENT_ERROR"

	IdLength = 12
)

var letterRunes = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func init() {
	rand.Seed(time.Now().UnixNano())
}

func randomId(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

type EventHead struct {
	Id     string `json:"id"`
	Action string `json:"action"`
	To     string `json:"to,omitempty"`
}

type Event struct {
	*EventHead
	Payload *jsoniter.RawMessage `json:"payload,omitempty"`
}

type EventNewHubCreated struct {
	*EventHead
	Payload NewHubPayload `json:"payload"`
}

type EventHubRemoved struct {
	*EventHead
	Payload HubRemovedPayload `json:"payload"`
}

type EventAllHubs struct {
	*EventHead
	Payload AllPayload `json:"payload"`
}

type EventError struct {
	*EventHead
	Payload ErrorPayload `json:"payload"`
}

type ErrorPayload struct {
	Info string `json:"info"`
}

type HubRemovedPayload struct {
	Name string `json:"name"`
}

type NewHubPayload struct {
	Name string `json:"name"`
}

type AllPayload struct {
	Hubs []string `json:"hubs"`
}

func ConsumeEvent(c *Client, event Event) {
	switch event.Action {
	case EVENT_NEW_HUB_REQUEST:
		consumeNewHubEvent(c, event)
	case EVENT_OFFER_CONNECTION:
		fallthrough
	case EVENT_ANSWER_CONNECTION:
		fallthrough
	case EVENT_CANDIDATE_CONNECTION:
		consumeDirectRawEvent(c, event)
	case EVENT_GET_HUBS:
		consumeGetHubs(c, event)
	}
}

func consumeNewHubEvent(c *Client, event Event) {
	var payload NewHubPayload
	if err := jsoniter.Unmarshal(*event.Payload, &payload); err != nil {
		log.Println("consumeNewHubEvent", err)
	}
	newHub := NewHub(payload.Name, c.hub.cluster)
	c.hub.cluster.Add(newHub)
	c.attachToHub(newHub)
	emitNewHubCreated(c, newHub.ID)
}

func consumeDirectRawEvent(c *Client, event Event) {
	addressee := c.hub.Get(event.To)
	if addressee == nil {
		clientNotFound(c, event.To, event.Id)
		return
	}
	bts, err := jsoniter.Marshal(event)
	if err != nil {
		log.Println("consumeDirectRawEvent", err)
		return
	}
	addressee.Send(bts)
}

func consumeGetHubs(c *Client, event Event) {
	bts, err := jsoniter.Marshal(EventAllHubs{
		EventHead: &EventHead{
			Id:     event.Id,
			Action: EVENT_GET_HUBS,
			To:     c.name,
		},
		Payload: AllPayload{
			Hubs: c.hub.cluster.All(),
		},
	})
	if err != nil {
		log.Println("consumeGetHubs", err)
		return
	}
	c.Send(bts)
}

func clientNotFound(c *Client, name string, id string) {
	bts, err := jsoniter.Marshal(EventError{
		EventHead: &EventHead{
			Id:     id,
			Action: EVENT_ERROR,
			To:     c.name,
		},
		Payload: ErrorPayload{
			Info: fmt.Sprintf("Client with name %s not found in your space", name),
		},
	})
	if err != nil {
		log.Println("clientNotFound", err)
		return
	}
	c.Send(bts)
}

func emitNewHubCreated(c *Client, name string) {
	bts, err := jsoniter.Marshal(EventNewHubCreated{
		EventHead: &EventHead{
			Id:     randomId(IdLength),
			Action: EVENT_NEW_HUB_CREATED,
			To:     TO_EVERYONE,
		},
		Payload: NewHubPayload{
			Name: name,
		},
	})
	if err != nil {
		log.Println("emitNewHubCreated", err)
		return
	}
	c.hub.cluster.General.Emit(bts)
}

func consumeHubRemoved(cluster *Cluster, event EventHubRemoved) {
	bts, err := jsoniter.Marshal(event)
	if err != nil {
		log.Println("consumeHubRemoved", err)
		return
	}
	cluster.General.Emit(bts)
}

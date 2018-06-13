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

	EVENT_NEW_HUB_REQUEST  = "EVENT_NEW_HUB_REQUEST"
	EVENT_HUB_CONNECT      = "EVENT_HUB_CONNECT"
	EVENT_CLIENT_CONNECTED = "EVENT_CLIENT_CONNECTED"
	EVENT_NEW_HUB_CREATED  = "EVENT_NEW_HUB_CREATED"
	EVENT_HUB_REMOVED      = "EVENT_HUB_REMOVED"
	EVENT_CLIENT_REMOVED   = "EVENT_CLIENT_REMOVED"
	EVENT_GET_CLIENTS      = "EVENT_GET_CLIENTS"
	EVENT_GET_HUBS         = "EVENT_GET_HUBS"

	EVENT_OFFER_CONNECTION     = "EVENT_OFFER_CONNECTION"
	EVENT_ANSWER_CONNECTION    = "EVENT_ANSWER_CONNECTION"
	EVENT_CANDIDATE_CONNECTION = "EVENT_CANDIDATE_CONNECTION"

	EVENT_CLIENT_REPLY_REQUEST  = "EVENT_CLIENT_REPLY_REQUEST"
	EVENT_CLIENT_REPLY_RESPONSE = "EVENT_CLIENT_REPLY_RESPONSE"

	EVENT_ERROR   = "EVENT_ERROR"
	EVENT_CONFIRM = "EVENT_CONFIRM"

	IdLength     = 12
	ReplyTimeout = time.Second * 5
)

var letterRunes = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
var eventsKeyQueue = NewKeyQueue()

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

type EventClientConnected struct {
	*EventHead
	Payload ClientConnectPayload `json:"payload"`
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

type EventConfirm struct {
	*EventHead
	Payload ConfirmPayload `json:"payload"`
}

type EventClientRemoved struct {
	*EventHead
	Payload ClientRemovedPayload `json:"payload"`
}

type EventHubConnect struct {
	*EventHead
	Payload HubConnectPayload `json:"payload"`
}

type EventGetClients struct {
	*EventHead
	Payload GetClientsPayload `json:"payload"`
}

type ErrorPayload struct {
	Info string `json:"info"`
}

type ConfirmPayload struct {
	Success bool `json:"success"`
}

type HubRemovedPayload struct {
	Name string `json:"name"`
}

type HubConnectPayload struct {
	Name string `json:"name"`
}

type GetClientsPayload struct {
	Clients []string `json:"clients"`
}

type NewHubPayload struct {
	Name string `json:"name"`
}

type ClientRemovedPayload struct {
	Name string `json:"name"`
}

type ClientConnectPayload struct {
	Name string `json:"name"`
}

type AllPayload struct {
	Hubs []string `json:"hubs"`
}

type HubClientPayload struct {
	Clients []string `json:"clients"`
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
	case EVENT_HUB_CONNECT:
		consumeHubConnectEvent(c, event)
	case EVENT_GET_CLIENTS:
		consumeGetClients(c, event)
	case EVENT_CLIENT_REPLY_REQUEST:
		consumeClientReplyRequest(c, event)
	case EVENT_CLIENT_REPLY_RESPONSE:
		consumeClientReplyResponse(c, event)
	}
}

func consumeNewHubEvent(c *Client, event Event) {
	var payload NewHubPayload
	if err := jsoniter.Unmarshal(*event.Payload, &payload); err != nil {
		log.Println("consumeNewHubEvent", err)
	}
	if h := c.Hub.cluster.Get(payload.Name); h != nil {
		hubAlreadyExist(c, payload.Name, event.Id)
		return
	}
	newHub := NewHub(payload.Name, c.Hub.cluster)
	c.Hub.cluster.Add(newHub)
	newHub.Add(c)
	emitNewHubCreated(c, newHub.ID)
	confirmAction(c, event.Id)
}

func consumeHubConnectEvent(c *Client, event Event) {
	var payload HubConnectPayload
	if err := jsoniter.Unmarshal(*event.Payload, &payload); err != nil {
		log.Println("consumeHubConnectEvent", err)
	}
	hub := c.Hub.cluster.Get(payload.Name)
	if hub == nil {
		hubNotFound(c, payload.Name, event.Id)
		return
	}
	hub.Add(c)
	emitClientConnected(c, hub)
	confirmAction(c, event.Id)
}

func consumeDirectRawEvent(c *Client, event Event) {
	addressee := c.Hub.Get(event.To)
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
			To:     c.Name,
		},
		Payload: AllPayload{
			Hubs: c.Hub.cluster.All(),
		},
	})
	if err != nil {
		log.Println("consumeGetHubs", err)
		return
	}
	c.Send(bts)
}

func consumeGetClients(c *Client, event Event) {
	bts, err := jsoniter.Marshal(EventGetClients{
		EventHead: &EventHead{
			Id:     event.Id,
			Action: EVENT_GET_CLIENTS,
			To:     c.Name,
		},
		Payload: GetClientsPayload{
			Clients: c.Hub.All(),
		},
	})
	if err != nil {
		log.Println("consumeGetClients", err)
		return
	}
	c.Send(bts)
}

func consumeClientReplyResponse(c *Client, event Event) {
	if event.To == "" {
		clientNotFound(c, "", event.Id)
		return
	}
	ch := eventsKeyQueue.Get(event.Id + event.To)
	if ch == nil {
		clientNotWaiting(c, event.To, event.Id)
		return
	}
	ch <- event
	confirmAction(c, event.Id)
}

func consumeClientReplyRequest(c *Client, event Event) {
	if event.To == "" {
		clientNotFound(c, "", event.Id)
		return
	}
	responder := c.Hub.Get(event.To)
	bts, err := jsoniter.Marshal(event)
	if err != nil {
		log.Println("consumeClientReplyRequest", err)
		return
	}
	reply := make(chan Event)
	eventsKeyQueue.Set(event.Id+c.Name, reply)
	responder.Send(bts)
	select {
	case e := <-reply:
		bts, err := jsoniter.Marshal(e)
		if err != nil {
			log.Println("consumeClientReplyRequest", err)
			return
		}
		c.Send(bts)
	case <-time.After(ReplyTimeout):
		clientAnswerTimeout(c, event.To, event.Id)
	}
}

func clientNotFound(c *Client, name string, id string) {
	bts, err := jsoniter.Marshal(EventError{
		EventHead: &EventHead{
			Id:     id,
			Action: EVENT_ERROR,
			To:     c.Name,
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

func clientAnswerTimeout(c *Client, name string, id string) {
	bts, err := jsoniter.Marshal(EventError{
		EventHead: &EventHead{
			Id:     id,
			Action: EVENT_ERROR,
			To:     c.Name,
		},
		Payload: ErrorPayload{
			Info: fmt.Sprintf("Client with name %s didn't repond", name),
		},
	})
	if err != nil {
		log.Println("clientAnswerTimeout", err)
		return
	}
	c.Send(bts)
}

func clientNotWaiting(c *Client, name string, id string) {
	bts, err := jsoniter.Marshal(EventError{
		EventHead: &EventHead{
			Id:     id,
			Action: EVENT_ERROR,
			To:     c.Name,
		},
		Payload: ErrorPayload{
			Info: fmt.Sprintf("Client %s is not waiting for response on message with id %s", name, id),
		},
	})
	if err != nil {
		log.Println("clientNotWaiting", err)
		return
	}
	c.Send(bts)
}

func hubNotFound(c *Client, hubId string, id string) {
	bts, err := jsoniter.Marshal(EventError{
		EventHead: &EventHead{
			Id:     id,
			Action: EVENT_ERROR,
			To:     c.Name,
		},
		Payload: ErrorPayload{
			Info: fmt.Sprintf("Hub with id %s not found", hubId),
		},
	})
	if err != nil {
		log.Println("clientNotFound", err)
		return
	}
	c.Send(bts)
}

func hubAlreadyExist(c *Client, hubId string, id string) {
	bts, err := jsoniter.Marshal(EventError{
		EventHead: &EventHead{
			Id:     id,
			Action: EVENT_ERROR,
			To:     c.Name,
		},
		Payload: ErrorPayload{
			Info: fmt.Sprintf("Hub with id %s already exist", hubId),
		},
	})
	if err != nil {
		log.Println("clientNotFound", err)
		return
	}
	c.Send(bts)
}

func confirmAction(c *Client, id string) {
	bts, err := jsoniter.Marshal(EventConfirm{
		EventHead: &EventHead{
			Id:     id,
			Action: EVENT_CONFIRM,
			To:     c.Name,
		},
		Payload: ConfirmPayload{
			Success: true,
		},
	})
	if err != nil {
		log.Println("confirmAction", err)
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
	c.Hub.cluster.General.Emit(bts)
}

func emitClientConnected(c *Client, hub *Hub) {
	bts, err := jsoniter.Marshal(EventClientConnected{
		EventHead: &EventHead{
			Id:     randomId(IdLength),
			Action: EVENT_CLIENT_CONNECTED,
			To:     TO_EVERYONE,
		},
		Payload: ClientConnectPayload{
			Name: c.Name,
		},
	})
	if err != nil {
		log.Println("emitNewHubCreated", err)
		return
	}
	hub.Emit(bts)
}

func getClientRemoved(name string) []byte {
	bts, err := jsoniter.Marshal(EventClientConnected{
		EventHead: &EventHead{
			Id:     randomId(IdLength),
			Action: EVENT_CLIENT_REMOVED,
			To:     TO_EVERYONE,
		},
		Payload: ClientConnectPayload{
			Name: name,
		},
	})
	if err != nil {
		log.Println("emitClientRemoved", err)
		return nil
	}
	return bts
}

func consumeHubRemoved(cluster *Cluster, event EventHubRemoved) {
	bts, err := jsoniter.Marshal(event)
	if err != nil {
		log.Println("consumeHubRemoved", err)
		return
	}
	cluster.General.Emit(bts)
}

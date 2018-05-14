package ws

import (
	"github.com/gorilla/websocket"
	"github.com/labstack/echo"
	"github.com/lempiy/Signaller/room"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	closeErrorCodes = []int{
		websocket.CloseAbnormalClosure,
		websocket.CloseNormalClosure,
		websocket.CloseInternalServerErr,
	}
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 30 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

func Handle(cluster *room.Cluster) echo.HandlerFunc {
	return func(c echo.Context) error {
		var hub *room.Hub
		var client *room.Client
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt)
		ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}

		space := c.QueryParam("space")
		if space == "" {
			ws.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(4001, "Connection space cannot be empty"),
			)
			ws.Close()
			return nil
		}

		name := c.QueryParam("name")
		if name == "" {
			ws.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(4001, "Player name cannot be empty"),
			)
			ws.Close()
			return nil
		}

		send := make(chan []byte)
		read := make(chan []byte)
		die := make(chan struct{})

		if hub = cluster.Get(space); hub == nil {
			hub = room.NewHub(space, cluster)
			cluster.Add(hub)
		}

		client = room.NewClient(send, read, die, name)
		hub.Add(client)
		deadRead := make(chan struct{})

		go func() {
			defer ws.Close()
			ws.SetReadDeadline(time.Now().Add(pongWait))
			ws.SetPongHandler(func(s string) error {
				ws.SetReadDeadline(time.Now().Add(pongWait))
				return nil
			})
			for {
				_, message, err := ws.ReadMessage()
				if err != nil {
					if websocket.IsCloseError(err, closeErrorCodes...) {
						ws.Close()
						client.Die()
						if hub.Length() == 0 {
							cluster.Remove(hub.ID)
							hub.Die()
						}
						deadRead <- struct{}{}
						return
					}
					ws.Close()
					client.Die()
					if hub.Length() == 0 {
						cluster.Remove(hub.ID)
						hub.Die()
					}
					deadRead <- struct{}{}
					return
				}
				read <- message
			}
		}()
		ticker := time.NewTicker(pingPeriod)
		for {
			select {
			case data := <-send:
				ws.SetWriteDeadline(time.Now().Add(writeWait))
				err := ws.WriteMessage(websocket.TextMessage, data)
				if err != nil {
					log.Println(err)
				}
			case <-deadRead:
				return err
			case <-ticker.C:
				ws.SetWriteDeadline(time.Now().Add(writeWait))
				err := ws.WriteMessage(websocket.PingMessage, []byte{})
				if err != nil {
					ws.Close()
					client.Die()
					if hub.Length() == 0 {
						cluster.Remove(hub.ID)
						hub.Die()
					}
					log.Println(err)
				}
			case <-interrupt:
				log.Println("Websocket server disconnect...")
				err := ws.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure,
						"Server's gone down"))
				ws.Close()
				return err
			}
		}
	}
}

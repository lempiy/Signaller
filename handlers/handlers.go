package handlers

import (
	"github.com/labstack/echo"
	"github.com/lempiy/Signaller/handlers/ws"
	"github.com/lempiy/Signaller/room"
)

//Run - inits and fills app router with handlers.
func Run(r *echo.Router, cluster *room.Cluster) {
	r.Add("GET", "/ws", ws.Handle(cluster))
}

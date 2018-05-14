package main

import (
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/lempiy/Signaller/handlers"
	"github.com/lempiy/Signaller/room"
	"os"
)

func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	cluster := room.NewCluster()
	PORT := os.Getenv("PORT")
	if PORT == "" {
		PORT = "4000"
	}
	r := e.Router()
	handlers.Run(r, cluster)
	e.Logger.Fatal(e.Start(":" + PORT))
}

package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/CiaranWoodward/broadcast_hub/server"
	"github.com/urfave/cli/v2"
)

func main() {
	//Using urfave/cli to make sensible CLI argument parsing
	app := &cli.App{
		Name:                   "server",
		Usage:                  "The broadcast_hub server, for accepting incoming client connections and enabling them to communicate",
		Action:                 runServer,
		UseShortOptionHandling: true,
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:     "port",
				Aliases:  []string{"p"},
				Usage:    "Listen on the given `PORT` for incoming TCP connections.",
				Required: true,
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

// Handle the top-level CLI arguments, start the parser
func runServer(c *cli.Context) error {
	port := c.Int("port")

	if port < 1 || port > 0xFFFF {
		log.Fatalf("PORT out of range: %d", port)
	}

	// TCP connect
	endpoint := fmt.Sprintf(":%d", port)
	ser := server.NewServer()
	listener, err := net.Listen("tcp", endpoint)
	if err != nil {
		log.Fatalf("Failed to listen on port %d", port)
	}
	ser.AddListener(listener)

	log.Printf("Successfully listening on port %d.", port)
	log.Println("Use Ctl-C to exit.")

	// Run until ctl-c
	quit := make(chan os.Signal, 2)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	return nil
}

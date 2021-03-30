/*
Basic CLI demonstrating the broadcast_hub client
*/
package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/CiaranWoodward/broadcast_hub/client"
	"github.com/CiaranWoodward/broadcast_hub/msg"
	"github.com/urfave/cli/v2"
)

// This entire file is
func main() {
	//Using urfave/cli to give sensible CLI argument parsing
	app := &cli.App{
		Name:                   "client",
		Usage:                  "The broadcast_hub client, for connecting to a server and communicating with other clients",
		Action:                 runClient,
		UseShortOptionHandling: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "server",
				Aliases:  []string{"s"},
				Usage:    "Connect to the broadcast_hub server at the provided `HOSTNAME`.",
				Required: true,
			},
			&cli.IntFlag{
				Name:     "port",
				Aliases:  []string{"p"},
				Usage:    "Connect to the given `PORT` of the broadcast_hub server.",
				Required: true,
			},
			&cli.IntFlag{
				Name:  "roger_no",
				Usage: "Create the given `COUNT` of dummy clients, which will respond back with a message whenever they are contacted",
				Value: 0,
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

// Handle the top-level CLI arguments, start the parser
func runClient(c *cli.Context) error {
	port := c.Int("port")
	servername := c.String("server")
	roger_no := c.Int("roger_no")

	if port < 1 || port > 0xFFFF {
		log.Fatalf("PORT out of range: %d", port)
	}

	// TCP connect
	endpoint := fmt.Sprintf("%s:%d", servername, port)
	con, err := net.Dial("tcp", endpoint)
	if err != nil {
		log.Fatal(err)
	}

	// Create dummy clients alongside
	createRogers(roger_no, endpoint)

	// Bind to client
	myClient := client.NewClient(con)

	// Get client ID & start up!
	cid, status := myClient.GetClientId()
	if status != msg.SUCCESS {
		log.Fatal(status)
	}
	log.Printf("Successfully connected to server %s, with CID %d.", endpoint, cid)

	startPrinter(myClient)
	startInteractive(myClient)

	return nil
}

func startPrinter(c *client.Client) {
	// Goroutine to print all incoming relays
	go func() {
		for {
			rx, ok := <-c.Relays
			if !ok {
				break
			}
			fmt.Printf("Rx from %d: %s\n", rx.Src, rx.Msg)
		}
	}()
}

func printHelp() {
	log.Println("Interactive Help:")
	log.Println(" getid")
	log.Println("\t- Get the ID of this client")
	log.Println(" list")
	log.Println("\t- Get the IDs of the other connected clients")
	log.Println(" relay <space seperated list of Client IDs> : <ASCII Message>")
	log.Println("\t- Send a message to the list of other Clients, via the hub.")
	log.Println("\t  Eg: relay 1 2 34 :Hello there!")
	log.Println(" quit")
}

func startInteractive(c *client.Client) {
	defer c.Close()

	printHelp()
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print(">")
		scanner.Scan()
		line := scanner.Text()
		split := strings.SplitN(line, " ", 2)
		if len(split) == 0 {
			continue
		}
		command := split[0]
		args := ""
		if len(split) == 2 {
			args = split[1]
		}

		switch command {
		case "getid":
			cid, status := c.GetClientId()
			if status != msg.SUCCESS {
				log.Printf("Error: %v", status)
			}
			log.Printf("My ID: %d\n", cid)

		case "list":
			cids, status := c.ListOtherClients()
			if status != msg.SUCCESS {
				log.Printf("Error: %v", status)
			}
			log.Printf("Other IDs: %v\n", cids)

		case "relay":
			cids, mesg, err := relayCommandParse(args)
			if err != nil {
				log.Printf("Parse Error: %v", err)
			}
			csm, status := c.RelayMessage(mesg, cids)
			if status != msg.SUCCESS {
				log.Printf("Error: %v", status)
			} else if len(csm) > 0 {
				log.Printf("Partial Error: %v", csm)
			} else {
				log.Println("Success!")
			}

		case "quit":
			return
		default:
			log.Printf("Unrecognised command \"%s\".\n", command)
		}
	}
}

func relayCommandParse(args string) (cids []msg.ClientId, mesg []byte, err error) {
	split := strings.SplitN(args, ":", 2)
	if len(split) == 2 {
		mesg = []byte(split[1])
	} else if len(split) == 0 {
		err = fmt.Errorf("relay command invalid format")
		return
	}

	// Convert the space seperate list into a ClientId Slice
	cids_string := strings.Split(split[0], " ")
	for _, cs := range cids_string {
		i, e := strconv.ParseUint(cs, 10, 64)
		if e != nil {
			err = e
			return
		}
		cids = append(cids, msg.ClientId(i))
	}
	return
}

func createRogers(n int, ep string) {
	for i := 0; i < n; i++ {
		go func() {
			con, err := net.Dial("tcp", ep)
			if err != nil {
				log.Printf("Failed to create Roger #%d: %v", i, err)
				return
			}

			// Bind to client
			myClient := client.NewClient(con)
			cid, status := myClient.GetClientId()
			if status != msg.SUCCESS {
				log.Fatal(status)
			}
			log.Printf("Successfully started Roger %d", cid)

			// Loop forever responding to messages
			for {
				rx, ok := <-myClient.Relays
				if !ok {
					break
				}
				src := rx.Src
				respm := fmt.Sprintf("Roger that %d - I am %d!", src, cid)
				go myClient.RelayMessage([]byte(respm), []msg.ClientId{src})
			}
		}()
	}
}

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

/// Append a new connection.
func (c *DataPasser) add(cs ClientServer) {
	c.Connections = append(c.Connections, cs)
}
func (c *DataPasser) addServer(id string, conn *websocket.Conn) {
	_, connection, index := c.find(id)
	connection.server = conn
	c.Connections[index] = *connection
}

/// Find the Connections by the key.
func (c *DataPasser) find(key string) (error, *ClientServer, int) {
	for i, c := range c.Connections {
		if c.clientKey == key {
			return nil, &c, i
		}
	}
	return errors.New("no connection has been found"), nil, 0
}

/// Add a client
func (c *DataPasser) addClient(key string, conn *websocket.Conn) {
	_, connection, index := c.find(key)
	connection.client = conn
	c.Connections[index] = *connection
}

type message struct {
	State   string            `json:"state"`
	Message string            `json:"message"`
	Data    map[string]string `json:"data"`
}

type DataPasser struct {
	Channel     chan ClientServer
	Connections []ClientServer
}

func (dp *DataPasser) wsEndpoint(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}

	m := message{}
	m.State = "ACCESS_GRANTED"
	response, _ := json.Marshal(m)
	err = ws.WriteMessage(1, response)
	if err != nil {
		log.Println(err)
	}
	log.Println("Client Connected")
	for {
		messageType, p, err := ws.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		log.Printf("Message received: %s", string(p))
		var m message
		err = json.Unmarshal(p, &m)
		if err != nil {
			return
		}
		cs := ClientServer{}
		if m.State == "REGISTERING_CLIENT" {
			fmt.Println("Spawning new container")
			id := m.Data["key"]
			cs.clientKey = id
			cs.client = ws
			spawn(true, id)
			dp.Channel <- cs
		}
		if m.State == "REGISTERING_RECORDING" {
			cs.clientKey = m.Data["key"]
			cs.server = ws
			fmt.Println(m.Data)
			m := message{State: "START_RECORDING"}
			bytes, err := json.Marshal(m)
			if err != nil {
				log.Println(err)
			}
			err = ws.WriteMessage(messageType, bytes)
			if err != nil {
				log.Println(err)
			}
			dp.Channel <- cs
		}

		if m.State == "RECORDING_STARTED" {
			fmt.Println("Recording has been started in docker container")
		}

	}
}

func main() {
	dp := DataPasser{
		Channel:     make(chan ClientServer),
		Connections: []ClientServer{},
	}
	go dp.mergeConnections()
	http.HandleFunc("/ws", dp.wsEndpoint)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func (dp *DataPasser) mergeConnections() {
	fmt.Println("Listening for new connections")

	for {
		select {
		case conn := <-dp.Channel:
			_, c, _ := dp.find(conn.clientKey)
			if c == nil {
				dp.add(conn)
			} else {
				// If the server is already registered this is probably the client.
				if c.server != nil {
					dp.addClient(conn.clientKey, conn.client)
				}
			}
			fmt.Println("The current size of the connections is: ", len(dp.Connections))
			fmt.Println("New connection received", conn.clientKey)
		}
	}
}

type ClientServer struct {
	clientKey string
	client    *websocket.Conn
	server    *websocket.Conn
}

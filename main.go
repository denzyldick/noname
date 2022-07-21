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
func (dp *DataPasser) add(cs ClientServer) {
	dp.Connections = append(dp.Connections, cs)
}
func (dp *DataPasser) addServer(id string, conn *websocket.Conn) {
	_, connection, index := dp.find(id)
	connection.server = conn
	dp.Connections[index] = *connection
}

/// Find the Connections by the key.
func (dp *DataPasser) find(key string) (error, *ClientServer, int) {
	for i, c := range dp.Connections {
		if c.clientKey == key {
			return nil, &c, i
		}
	}
	return errors.New("no connection has been found"), nil, 0
}

/// Add a client
func (dp *DataPasser) addClient(key string, conn *websocket.Conn) {
	_, connection, index := dp.find(key)
	connection.client = conn
	dp.Connections[index] = *connection
}

type message struct {
	State   string                 `json:"state"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}

type Broadcast struct {
	clientKey string
	Message   []byte
}

type DataPasser struct {
	Channel     chan ClientServer
	Connections []ClientServer
	Broadcast   chan Broadcast
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
			log.Println(err)
			return
		}
		cs := ClientServer{}
		if m.State == "REGISTERING_CLIENT" {
			fmt.Println("Spawning new container")
			id := fmt.Sprint(m.Data["key"])
			cs.clientKey = id
			cs.client = ws
			dp.Channel <- cs
		}
		if m.State == "REGISTERING_RECORDING" {
			cs.clientKey = fmt.Sprint(m.Data["key"])
			cs.server = ws
			fmt.Println(m.Data)
			dp.Channel <- cs
		}

		if m.State == "RECORDING_STARTED" {
			fmt.Println("Recording has been started in docker container")
		}

		if m.State == "BROADCAST" {
			bytes, err := json.Marshal(m)

			if err != nil {
				err := ws.WriteMessage(messageType, []byte("Your message is not a JSON object."))
				if err != nil {
					return
				}
			}
			dp.Broadcast <- Broadcast{
				clientKey: fmt.Sprint(m.Data["key"]),
				Message:   bytes,
			}
		}

	}
}

func main() {
	dp := DataPasser{
		Channel:     make(chan ClientServer),
		Connections: []ClientServer{},
		Broadcast:   make(chan Broadcast),
	}
	go dp.mergeConnections()
	http.HandleFunc("/ws", dp.wsEndpoint)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func (dp *DataPasser) mergeConnections() {
	fmt.Println("Listening for new connections")

	for {
		select {
		case Broadcast := <-dp.Broadcast:
			err, c, _ := dp.find(Broadcast.clientKey)
			fmt.Println(string(Broadcast.Message))
			fmt.Println(err)
			err = c.server.WriteMessage(1, Broadcast.Message)
			if err != nil {
				return
			}
			err = c.client.WriteMessage(1, Broadcast.Message)
			if err != nil {
				return
			}
		case conn := <-dp.Channel:
			_, c, _ := dp.find(conn.clientKey)
			if c == nil {
				dp.add(conn)
				go spawn(true, conn.clientKey)
			} else {
				// If the server is already registered this is probably the client.
				if c.server == nil {
					dp.addServer(conn.clientKey, conn.server)
					/// Start
					m := message{State: "START_RECORDING"}
					bytes, err := json.Marshal(m)
					if err != nil {
						log.Println(err)
					}
					err = conn.server.WriteMessage(1, bytes)
					if err != nil {
						log.Println(err)
					}
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

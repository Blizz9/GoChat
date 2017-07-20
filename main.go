package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"strconv"

	"github.com/gorilla/websocket"
)

const (
	port         = 8080
	bufferSize   = 1024
	chatLogCount = 100
)

type wsMessage struct {
	Type    string
	Content wsMessageContent
}

type wsMessageContent struct {
	Username  string
	Timestamp int64
	Message   string
}

var connections []*websocket.Conn

func main() {
	connections = []*websocket.Conn{}

	http.HandleFunc("/chat", chatHandler)
	http.HandleFunc("/chat/log", chatLogHandler)
	http.HandleFunc("/chat/ws", wsHandler)

	panic(http.ListenAndServe(":"+strconv.Itoa(port), nil))
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadFile("chat.html")
	if err != nil {
		http.Error(w, "Could not open chat.html file.", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "%s", body)
}

func chatLogHandler(w http.ResponseWriter, r *http.Request) {
	messages := retreiveMessages(chatLogCount)

	json, err := json.Marshal(messages)
	if err != nil {
		http.Error(w, "Unable to format JSON response.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Origin") != "http://"+r.Host {
		http.Error(w, "The request must originate from the host.", http.StatusBadRequest)
		return
	}

	conn, err := websocket.Upgrade(w, r, w.Header(), bufferSize, bufferSize)
	if err != nil {
		http.Error(w, "Failed opening the websocket connection.", http.StatusInternalServerError)
		return
	}

	connections = append(connections, conn)
	fmt.Println(len(connections))

	go wsMessageHandler(conn)
}

func wsMessageHandler(conn *websocket.Conn) {
	for {
		message := wsMessage{}

		messageType, data, err := conn.ReadMessage()
		if messageType == -1 {
			fmt.Println("Websocket closed by client.", err)
			removeConnection(conn)
			return
		} else if err != nil {
			fmt.Println("Error reading websocket message.", err)
			removeConnection(conn)
			return
		}

		err = json.Unmarshal(data, &message)
		if err != nil {
			fmt.Println("Error parsing websocket message.", err)
		} else {
			fmt.Printf("Recieved websocket message: %#v\n", message)

			storeMessage(message.Content)

			for _, otherConn := range connections {
				if conn != otherConn {
					otherConn.WriteJSON(message)
				}
			}
		}
	}
}

func removeConnection(connToRemove *websocket.Conn) {
	for i, conn := range connections {
		if connToRemove == conn {
			connections = append(connections[:i], connections[i+1:]...)
			fmt.Println(len(connections))
			break
		}
	}
}

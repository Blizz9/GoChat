package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"strconv"

	"github.com/gorilla/websocket"
)

const (
	port       = 80
	securePort = 443
	bufferSize = 1024
)

type wsMessage struct {
	Username  string
	Timestamp int64
	Message   string
}

// all of the live client connections will be stored here
var connections []*websocket.Conn

func main() {
	connections = []*websocket.Conn{}

	//http.HandleFunc("/chat", chatHandler)
	//http.HandleFunc("/chat/log", chatLogHandler)
	//http.HandleFunc("/chat/ws", wsHandler)

	go http.ListenAndServe(":"+strconv.Itoa(port), http.HandlerFunc(redirect))

	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/", index)
	serveMux.HandleFunc("/chat", chatHandler)
	serveMux.HandleFunc("/chat/log", chatLogHandler)
	serveMux.HandleFunc("/chat/ws", wsHandler)
	http.ListenAndServeTLS(":"+strconv.Itoa(securePort), "cert.pem", "key.pem", serveMux)

	//go http.ListenAndServe(":80", http.HandlerFunc(redirect))
	//panic(http.ListenAndServe(":"+strconv.Itoa(port), nil))

	//panic(http.ListenAndServe(":"+strconv.Itoa(port), http.HandlerFunc(redirect)))
}

// starting point for clients, serves up the chat.html page
func chatHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadFile("chat.html")
	if err != nil {
		http.Error(w, "Could not open chat.html file.", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "%s", body)
}

// used by the client to receive the past chat messages, pulled from datastore
func chatLogHandler(w http.ResponseWriter, r *http.Request) {
	messages := retreiveMessages()

	json, err := json.Marshal(messages)
	if err != nil {
		http.Error(w, "Unable to format JSON response.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

// web socket endpoint, upgrades the connection to a web socket connection
func wsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Origin") != "https://"+r.Host {
		http.Error(w, "The request must originate from the host.", http.StatusBadRequest)
		return
	}

	conn, err := websocket.Upgrade(w, r, w.Header(), bufferSize, bufferSize)
	if err != nil {
		http.Error(w, "Failed opening the websocket connection.", http.StatusInternalServerError)
		return
	}

	connections = append(connections, conn)
	fmt.Println("New websocket connection opened to client at " + conn.RemoteAddr().String())

	go wsMessageHandler(conn)
}

// each web socket message from the client is handled here
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

			storeMessage(message)

			// broadcast the message to the other clients
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
			fmt.Println("Websocket connection closed to client at " + connToRemove.RemoteAddr().String())
			break
		}
	}
}

func redirect(w http.ResponseWriter, r *http.Request) {
	target := "https://" + r.Host + r.URL.Path
	if len(r.URL.RawQuery) > 0 {
		target += "?" + r.URL.RawQuery
	}
	log.Printf("redirect to: %s", target)
	http.Redirect(w, r, target, http.StatusTemporaryRedirect)
}

func index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		log.Printf("404: %s", r.URL.String())
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "chat.html")
}

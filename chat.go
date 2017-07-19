package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"

	"github.com/gorilla/websocket"
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

func main() {
	svc := dynamodb.New(session.New(&aws.Config{Region: aws.String("us-east-1")}))

	result, err := svc.ListTables(&dynamodb.ListTablesInput{})
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Tables:")
	for _, table := range result.TableNames {
		fmt.Println(*table)
	}

	// tableName := "GoChat"

	// parameter := &dynamodb.GetItemInput{
	// 	Key: map[string]*dynamodb.AttributeValue{
	// 		"Username": {
	// 			S: aws.String("Nick"),
	// 		},
	// 		"Timestamp": {
	// 			N: aws.String("1500486387"),
	// 		},
	// 	},
	// 	TableName: &tableName,
	// }

	// response, err := svc.GetItem(parameter)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }

	// fmt.Println(response)

	//name := response["name"].S

	// response, err := Dyna.Db.GetItem(parameter)
	// if err != nil {
	// 	return nil, err
	// }

	// name := response["name"].S
	// return name, nil

	params := &dynamodb.ScanInput{
		TableName: aws.String("GoChat"),
		Limit:     aws.Int64(3),
	}

	response, err := svc.Scan(params)
	if err != nil {
		fmt.Println(err)
		return
	}

	items := []wsMessageContent{}

	err = dynamodbattribute.UnmarshalListOfMaps(response.Items, &items)
	if err != nil {
		fmt.Println(err)
		return
	}

	for i, item := range items {
		fmt.Printf("%d:  UserID: %s, Time: %d Msg: %s\n", i, item.Username, item.Timestamp, item.Message)
	}

	fmt.Println(response)

	http.HandleFunc("/chat", chatHandler)
	http.HandleFunc("/chat/log", chatLogHandler)
	http.HandleFunc("/chat/ws", wsHandler)

	panic(http.ListenAndServe(":8080", nil))
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
	svc := dynamodb.New(session.New(&aws.Config{Region: aws.String("us-east-1")}))

	scanInput := &dynamodb.ScanInput{
		TableName: aws.String("GoChat"),
		Limit:     aws.Int64(3),
	}

	scan, err := svc.Scan(scanInput)
	if err != nil {
		http.Error(w, "Unable to scan database.", http.StatusInternalServerError)
		return
	}

	messages := []wsMessageContent{}
	err = dynamodbattribute.UnmarshalListOfMaps(scan.Items, &messages)
	if err != nil {
		http.Error(w, "Unable to parse database items.", http.StatusInternalServerError)
		return
	}

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

	conn, err := websocket.Upgrade(w, r, w.Header(), 1024, 1024)
	if err != nil {
		http.Error(w, "Failed opening the websocket connection.", http.StatusInternalServerError)
		return
	}

	go wsMessageHandler(conn)
}

func wsMessageHandler(conn *websocket.Conn) {
	for {
		message := wsMessage{}

		err := conn.ReadJSON(&message)
		if err != nil {
			fmt.Println("Error parsing websocket message.", err)
		}

		fmt.Printf("Recieved websocket message: %#v\n", message.Content.Timestamp)

		svc := dynamodb.New(session.New(&aws.Config{Region: aws.String("us-east-1")}))

		writeInput := &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]*dynamodb.WriteRequest{
				"GoChat": {
					&dynamodb.WriteRequest{
						PutRequest: &dynamodb.PutRequest{
							Item: map[string]*dynamodb.AttributeValue{
								"Username": {
									S: aws.String(message.Content.Username),
								},
								"Timestamp": {
									N: aws.String(strconv.FormatInt(message.Content.Timestamp, 10)),
								},
								"Message": {
									S: aws.String(message.Content.Message),
								},
							},
						},
					},
				},
			},
		}

		_, err = svc.BatchWriteItem(writeInput)
		if err != nil {
			fmt.Println("Unable to properly write to database.", err)
		}
	}
}

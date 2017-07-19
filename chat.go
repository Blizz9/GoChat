package main

import (
	"fmt"
	"io/ioutil"
	"net/http"

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
	Message string
}

type dbItem struct {
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

	// params := &dynamodb.BatchWriteItemInput{
	// 	RequestItems: map[string][]*dynamodb.WriteRequest{
	// 		"GoChat": {
	// 			&dynamodb.WriteRequest{
	// 				PutRequest: &dynamodb.PutRequest{
	// 					Item: map[string]*dynamodb.AttributeValue{
	// 						"Username": {
	// 							S: aws.String("Nick"),
	// 						},
	// 						"Timestamp": {
	// 							N: aws.String(strconv.FormatInt(time.Now().UTC().Unix(), 10)),
	// 						},
	// 						"Message": {
	// 							S: aws.String("Hello Again!"),
	// 						},
	// 					},
	// 				},
	// 			},
	// 		},
	// 	},
	// }

	// writeResult, err := svc.BatchWriteItem(params)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// fmt.Println(writeResult.GoString())

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

	items := []dbItem{}

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
		fmt.Println("Could not open chat.html file.", err)
	}
	fmt.Fprintf(w, "%s", body)
}

func chatLogHandler(w http.ResponseWriter, r *http.Request) {
	// body, err := ioutil.ReadFile("chat.html")
	// if err != nil {
	// 	fmt.Println("Could not open chat.html file.", err)
	// }
	// fmt.Fprintf(w, "%s", body)
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Origin") != "http://"+r.Host {
		http.Error(w, "The request must originate from the host.", http.StatusBadRequest)
		return
	}

	conn, err := websocket.Upgrade(w, r, w.Header(), 1024, 1024)
	if err != nil {
		http.Error(w, "Failed opening the websocket connection.", http.StatusInternalServerError)
	}

	go wsMessageHandler(conn)
}

func wsMessageHandler(conn *websocket.Conn) {
	for {
		msg := wsMessage{}

		err := conn.ReadJSON(&msg)
		if err != nil {
			fmt.Println("Error parsing websocket message.", err)
		}

		fmt.Printf("Recieved websocket message: %#v\n", msg.Content.Message)

		if err = conn.WriteJSON(msg); err != nil {
			fmt.Println(err)
		}
	}
}

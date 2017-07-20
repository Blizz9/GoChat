package main

import (
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

const availabilityZoneName = "us-east-1"

func storeMessage(message wsMessageContent) {
	svc := dynamodb.New(session.New(&aws.Config{Region: aws.String(availabilityZoneName)}))

	writeInput := &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]*dynamodb.WriteRequest{
			"GoChat": {
				&dynamodb.WriteRequest{
					PutRequest: &dynamodb.PutRequest{
						Item: map[string]*dynamodb.AttributeValue{
							"Username": {
								S: aws.String(message.Username),
							},
							"Timestamp": {
								N: aws.String(strconv.FormatInt(message.Timestamp, 10)),
							},
							"Message": {
								S: aws.String(message.Message),
							},
						},
					},
				},
			},
		},
	}

	_, err := svc.BatchWriteItem(writeInput)
	if err != nil {
		fmt.Println("Unable to properly write to database.", err)
		return
	}
}

func retreiveMessages(count int) []wsMessageContent {
	svc := dynamodb.New(session.New(&aws.Config{Region: aws.String(availabilityZoneName)}))

	scanInput := &dynamodb.ScanInput{
		TableName: aws.String("GoChat"),
		Limit:     aws.Int64(int64(count)),
	}

	scan, err := svc.Scan(scanInput)
	if err != nil {
		fmt.Println("Unable to scan database.", err)
		return nil
	}

	messages := []wsMessageContent{}
	err = dynamodbattribute.UnmarshalListOfMaps(scan.Items, &messages)
	if err != nil {
		fmt.Println("Unable to parse database items.", err)
		return nil
	}

	return messages
}

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

// NOTE: the standard AWS credentials file at ~/.aws/credentials is required for this to work
//       also a table with the proper schema is required in DynamoDB (is not created for this exercise)

func storeMessage(message wsMessage) {
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

func retreiveMessages() []wsMessage {
	svc := dynamodb.New(session.New(&aws.Config{Region: aws.String(availabilityZoneName)}))

	scanInput := &dynamodb.ScanInput{
		TableName: aws.String("GoChat"),
	}

	scan, err := svc.Scan(scanInput)
	if err != nil {
		fmt.Println("Unable to scan database.", err)
		return nil
	}

	messages := []wsMessage{} // notice db schema matches messages format, convention over configuration
	err = dynamodbattribute.UnmarshalListOfMaps(scan.Items, &messages)
	if err != nil {
		fmt.Println("Unable to parse database items.", err)
		return nil
	}

	return messages
}

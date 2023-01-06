package dbtask

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type QueryHistories struct {
	Item []QueryHistory
}

type QueryHistory struct {
	RoomId    string `dynamodbav:"roomId"`
	Timestamp int64  `dynamodbav:"timestamp"`
	Comment   string `dynamodbav:"comment"`
	Date      string `dynamodbav:"date"`
	DebtorId  string `dynamodbav:"debtorId"`
	Amount    int64  `dynamodbav:"amount"`
	MessageId string `dynamodbav:"messageId"`
}

func PutQueryHistory(db *dynamodb.DynamoDB, ID string, queryItem *QueryHistory) error {
	inputAV, err := dynamodbattribute.MarshalMap(queryItem)
	if err != nil {
		return err
	}
	input := &dynamodb.PutItemInput{
		TableName: aws.String("lineServiceSeisanQueryHistory"),
		Item:      inputAV,
	}
	_, err = db.PutItem(input)
	if err != nil {
		return err
	}
	return nil
}

// get query history
func GetQueryHistory(db *dynamodb.DynamoDB, ID string, historyItem *QueryHistories) error {
	getParamQuery := &dynamodb.QueryInput{
		TableName:              aws.String("lineServiceSeisanQueryHistory"),
		KeyConditionExpression: aws.String("#roomId = :roomId"),
		ExpressionAttributeNames: map[string]*string{
			"#roomId":    aws.String("roomId"),
			"#timestamp": aws.String("timestamp"),
			"#comment":   aws.String("comment"),
			"#date":      aws.String("date"),
			"#amount":    aws.String("amount"),
			"#debtorId":  aws.String("debtorId"),
			"#messageId": aws.String("messageId"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":roomId": {
				S: aws.String(ID),
			},
		},
		ProjectionExpression: aws.String("#roomId, #timestamp, #comment, #date, #amount, #debtorId, #messageId"),
	}
	dbResQuery, err := db.Query(getParamQuery)
	if err != nil {
		return err
	}
	for _, v := range dbResQuery.Items {
		p := QueryHistory{}
		err = dynamodbattribute.UnmarshalMap(v, &p)
		historyItem.Item = append(historyItem.Item, p)
	}
	if err != nil {
		return err
	}
	return err
}
func DeleteQueryHistory(db *dynamodb.DynamoDB, ID string, timestamp string) error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String("lineServiceSeisanQueryHistory"),
		Key: map[string]*dynamodb.AttributeValue{
			"roomId": {
				S: aws.String(ID),
			},
			"timestamp": {
				N: aws.String(timestamp),
			},
		},
		ReturnConsumedCapacity:      aws.String("NONE"),
		ReturnItemCollectionMetrics: aws.String("NONE"),
		ReturnValues:                aws.String("NONE"),
	}
	_, err := db.DeleteItem(input)
	if err != nil {
		return err
	}
	return nil
}

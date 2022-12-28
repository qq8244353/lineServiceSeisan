package task

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type RoomSetting struct {
	RoomId     string `dynamodbav:"roomId"`
	UserName1  string `dynamodbav:"userName1"`
	UserName2  string `dynamodbav:"userName2"`
	UserId1    string `dynamodbav:"userId1"`
	UserId2    string `dynamodbav:"userId2"`
	SeisanDone bool   `dynamodbav:"seisanDone"`
	QueryCnt   int64  `dynamodbav:"queryCnt"`
	PaymentDue int64  `dynamodbav:"paymentDue"`
}

func GetRoomSetting(db *dynamodb.DynamoDB, ID string, settingItem *RoomSetting) error {
	//get roomSetting
	getParam := &dynamodb.GetItemInput{
		TableName: aws.String("lineServiceSeisanRoomSetting"),
		Key: map[string]*dynamodb.AttributeValue{
			"roomId": {
				S: aws.String(ID),
			},
		},
	}
	dbRes, err := db.GetItem(getParam)
	if err != nil {
		return err
	}
	err = dynamodbattribute.UnmarshalMap(dbRes.Item, &settingItem)
	if err != nil {
		return err
	}
	return nil
}

func PutRoomSetting(db *dynamodb.DynamoDB, ID string, settingItem *RoomSetting) error {
	inputAV, err := dynamodbattribute.MarshalMap(settingItem)
	if err != nil {
		log.Fatal(err)
	}
	input := &dynamodb.PutItemInput{
		TableName: aws.String("lineServiceSeisanRoomSetting"),
		Item:      inputAV,
	}
	_, err = db.PutItem(input)
	if err != nil {
		return err
	}
	return nil
}

func UpdateDone(db *dynamodb.DynamoDB, roomId string, b bool) error {
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String("lineServiceSeisanRoomSetting"),
		Key: map[string]*dynamodb.AttributeValue{
			"roomId": {
				S: aws.String(roomId),
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#target": aws.String("seisanDone"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":newState": {
				BOOL: aws.Bool(b),
			},
		},
		UpdateExpression: aws.String("set #target = :newState"),
	}
	_, err := db.UpdateItem(input)
	if err != nil {
		return err
	}
	return nil
}

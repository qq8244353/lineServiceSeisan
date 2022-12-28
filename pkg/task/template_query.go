package task

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type TemplateQueries struct {
	Item []TemplateQuery
}

type TemplateQuery struct {
	RoomId   string `dynamodbav:"roomId"`
	Name     string `dynamodbav:"name"`
	DebtorId string `dynamodbav:"debtorId"`
	Amount   int64  `dynamodbav:"amount"`
}

func GetTemplateQuery(db *dynamodb.DynamoDB, ID string, name string, registeredItem *TemplateQuery) error {
	//get TemplateQuery
	getParam := &dynamodb.GetItemInput{
		TableName: aws.String("lineServiceSeisanTemplateQuery"),
		Key: map[string]*dynamodb.AttributeValue{
			"roomId": {
				S: aws.String(ID),
			},
			"name": {
				S: aws.String(name),
			},
		},
	}
	log.Print(ID)
	dbRes, err := db.GetItem(getParam)
	if err != nil {
		return err
	}
	err = dynamodbattribute.UnmarshalMap(dbRes.Item, &registeredItem)
	if err != nil {
		return err
	}
	return nil
}

// get registered all query
func GetAllTemplateQuery(db *dynamodb.DynamoDB, ID string, registeredItem *TemplateQueries) error {
	getParamQuery := &dynamodb.QueryInput{
		TableName:              aws.String("lineServiceSeisanTemplateQuery"),
		KeyConditionExpression: aws.String("#roomId = :roomId"),
		ExpressionAttributeNames: map[string]*string{
			"#roomId":   aws.String("roomId"),
			"#name":     aws.String("name"),
			"#debtorId": aws.String("debtorId"),
			"#amount":   aws.String("amount"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":roomId": {
				S: aws.String(ID),
			},
		},
		ProjectionExpression: aws.String("#roomId, #name, #debtorId, #amount"),
	}
	dbResQuery, err := db.Query(getParamQuery)
	if err != nil {
		return err
	}
	for _, v := range dbResQuery.Items {
		p := TemplateQuery{}
		err = dynamodbattribute.UnmarshalMap(v, &p)
		registeredItem.Item = append(registeredItem.Item, p)
	}
	if err != nil {
		return err
	}
	return err
}

func PutTemplateQuery(db *dynamodb.DynamoDB, ID string, registeringItem *TemplateQuery) error {
	inputAV, err := dynamodbattribute.MarshalMap(registeringItem)
	if err != nil {
		log.Fatal(err)
	}
	inputPut := &dynamodb.PutItemInput{
		TableName: aws.String("lineServiceSeisantask.TemplateQuery"),
		Item:      inputAV,
	}
	_, err = db.PutItem(inputPut)
	if err != nil {
		return err
	}
	return nil
}

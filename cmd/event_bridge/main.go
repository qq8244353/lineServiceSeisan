package main

import (
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/qq8244353/lineServiceSeisan/pkg/dbtask"
)

func handler() {
	sess, err := session.NewSession()
	db := dynamodb.New(sess)
	if err != nil {
		log.Fatal(err)
	}
	settingItem := []dbtask.PaymentSetting{}
	err = dbtask.GetAllRoomSettingOfPayment(db, &settingItem)
	log.Printf("%v", settingItem)
	if err != nil {
		log.Fatalf("%v", err)
	}
}
func main() {
	lambda.Start(handler)
}

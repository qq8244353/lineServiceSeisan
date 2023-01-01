package main

import (
	"log"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/qq8244353/lineServiceSeisan/pkg/dbtask"
	"github.com/qq8244353/lineServiceSeisan/pkg/msgtask"
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
	t := time.Now()
	today := int64(t.Day())
	month_end := int64(time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1).Day())
	log.Printf("%d, %d", today, month_end)
	for _, item := range settingItem {
		reqStruct := new(msgtask.Push)
		if today == item.PaymentDue || (today == month_end && today < item.PaymentDue) {
			reqStruct.To = item.RoomId
			reqStruct.Messages = []msgtask.Message{{Type: "text", Text: "本日は支払日です"}}
			msgtask.PushMessage(reqStruct)
		}
	}
}
func main() {
	lambda.Start(handler)
}

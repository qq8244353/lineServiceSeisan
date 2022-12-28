package qtask

import (
	"fmt"
	"log"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

func GetSetting() {
}
func UpdateUsername() {
}
func UpdateNotificationSetting() {
}
func UpdateHistorySetting() {
}
func UpdateClearanceDate() {
}
func RegisterTemplate() {
}
func RegisterQueryHistory() {
	//get room setting
	settingItem := dbtask.RoomSetting{}
	err := dbtask.GetRoomSetting(db, ID, &settingItem)
	if err != nil {
		log.Fatal(err)
	}
	//validate user id
	var debtorId string
	if settingItem.UserName1 == qs[2] {
		debtorId = settingItem.UserId1
	} else if settingItem.UserName2 == qs[2] {
		debtorId = settingItem.UserId2
	} else {
		errMessage := fmt.Sprintf("ユーザー名が正しくありません\n%s\n%s", settingItem.UserName1, settingItem.UserName2)
		reqStruct.Messages = []Message{{Type: "text", Text: errMessage}}
		//reply registered messages
		err := replyMessage(reqStruct)
		if err != nil {
			log.Fatal(err)
		}
		continue
	}
	//validate query cnt
	if settingItem.QueryCnt > 5 {
		errMessage := "クエリ登録の上限です\n"
		reqStruct.Messages = []Message{{Type: "text", Text: errMessage}}
		//reply registered messages
		err = replyMessage(reqStruct)
		if err != nil {
			log.Fatal(err)
		}
		continue
	}
	inputUpdate := &dynamodb.UpdateItemInput{
		TableName: aws.String("lineServiceSeisanRoomSetting"),
		Key: map[string]*dynamodb.AttributeValue{
			"roomId": {
				S: aws.String(ID),
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#target": aws.String("QueryCnt"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":newState": {
				N: aws.String(strconv.FormatInt(settingItem.QueryCnt+1, 10)),
			},
		},
		UpdateExpression: aws.String("set #target = :newState"),
	}
	_, err = db.UpdateItem(inputUpdate)
	if err != nil {
		log.Fatal(err)
	}
	//register query
	amount, err := strconv.ParseInt(qs[3], 10, 64)
	if err != nil {
		log.Fatal(err)
	}
	dbtask.PutTemplateQuery(db, ID, &dbtask.TemplateQuery{
		RoomId:   ID,
		Name:     qs[1],
		DebtorId: debtorId,
		Amount:   amount,
	})

	if err != nil {
		log.Fatal(err)
	}
	//reply registered messages
	reqStruct.Messages = []Message{{Type: "text", Text: "success"}}
	err = replyMessage(reqStruct)
	if err != nil {
		log.Fatal(err)
	}
}
func GetQueryHistory() {
}
func ExecuteTempmlate() {
}
func ExecuteClearance() {
}

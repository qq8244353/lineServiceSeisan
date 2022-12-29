package qtask

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	dbtask "github.com/qq8244353/lineServiceSeisan/pkg/dbtask"
	msgtask "github.com/qq8244353/lineServiceSeisan/pkg/msgtask"
)

func GetSetting() {
}
func UpdateUsername(db *dynamodb.DynamoDB, ID string, qs []string, reqStruct *msgtask.Response) {
	//get room setting
	settingItem := dbtask.RoomSetting{}
	err := dbtask.GetRoomSetting(db, ID, &settingItem)
	if err != nil {
		log.Fatal(err)
	}
	log.Print(qs)
	log.Printf("%d %d %d", len(qs[0]), len(qs[1]), len(qs[2]))
	if len(qs[2]) > 15 {
		reqStruct.Messages = []msgtask.Message{{Type: "text", Text: "ユーザー名は5文字以下にしてください"}}
		//reply registered messages
		err := msgtask.ReplyMessage(reqStruct)
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	if (qs[1] == settingItem.UserName1 && qs[2] == settingItem.UserName2) || (qs[1] == settingItem.UserName2 && qs[2] == settingItem.UserName1) {
		errMessage := fmt.Sprintf("ユーザーが重複しています\n%s\n%s", settingItem.UserName1, settingItem.UserName2)
		reqStruct.Messages = []msgtask.Message{{Type: "text", Text: errMessage}}
		//reply registered messages
		err := msgtask.ReplyMessage(reqStruct)
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	var toBeReplaced string
	if qs[1] == settingItem.UserName1 {
		toBeReplaced = "userName1"
	} else if qs[1] == settingItem.UserName2 {
		toBeReplaced = "userName2"
	} else {
		errMessage := fmt.Sprintf("ユーザー名が正しくありません\n%s\n%s", settingItem.UserName1, settingItem.UserName2)
		reqStruct.Messages = []msgtask.Message{{Type: "text", Text: errMessage}}
		//reply registered messages
		err := msgtask.ReplyMessage(reqStruct)
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String("lineServiceSeisanRoomSetting"),
		Key: map[string]*dynamodb.AttributeValue{
			"roomId": {
				S: aws.String(ID),
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#target": aws.String(toBeReplaced),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":newName": {
				S: aws.String(qs[2]),
			},
		},
		UpdateExpression: aws.String("set #target = :newName"),
	}
	_, err = db.UpdateItem(input)
	if err != nil {
		log.Fatal(err)
	}
	reqStruct.Messages = []msgtask.Message{{Type: "text", Text: "success"}}
}
func UpdateNotificationSetting() {
}
func UpdateHistorySetting() {
}
func UpdateClearanceDate() {
}
func RegisterQueryHistory(db *dynamodb.DynamoDB, ID string, qs []string, reqStruct *msgtask.Response, e msgtask.Hook_event) {
	dbtask.UpdateDone(db, ID, false)
	//get room setting
	settingItem := dbtask.RoomSetting{}
	err := dbtask.GetRoomSetting(db, ID, &settingItem)
	if err != nil {
		log.Fatal(err)
	}
	var debtorId string
	if qs[1] == settingItem.UserName1 {
		debtorId = settingItem.UserId1
	} else if qs[1] == settingItem.UserName2 {
		debtorId = settingItem.UserId2
	} else {
		errMessage := fmt.Sprintf("ユーザー名が正しくありません\n%s\n%s", settingItem.UserName1, settingItem.UserName2)
		reqStruct.Messages = []msgtask.Message{{Type: "text", Text: errMessage}}
		//reply registered messages
		err := msgtask.ReplyMessage(reqStruct)
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	amount, err := strconv.ParseInt(qs[2], 10, 64)
	if err != nil {
		log.Fatal(err)
	}
	t := time.Now()
	m := fmt.Sprintf("%d", t.Month())
	d := fmt.Sprintf("%d", t.Day())
	if len(m) == 1 {
		m = "  " + m
	}
	if len(d) == 1 {
		d = "  " + d
	}
	date := fmt.Sprintf("%s/%s", m, d)
	err = dbtask.PutQueryHistory(db, ID, &dbtask.QueryHistory{
		RoomId:    ID,
		Timestamp: e.Timestamp,
		DebtorId:  debtorId,
		Amount:    amount,
		Comment:   qs[3],
		Date:      date,
		MessageId: e.Message.Id,
	})
	if err != nil {
		log.Fatal(err)
	}
	// reqStruct.Messages = []Message{{Type: "text", Text: "success"}}
	//culculate debt for notify users current state
	historyItem := dbtask.QueryHistories{}
	err = dbtask.GetQueryHistory(db, ID, &historyItem)
	if err != nil {
		log.Fatal(err)
	}
	user1Debt := int64(0)
	for _, item := range historyItem.Item {
		//culculate debt sum
		if item.DebtorId == settingItem.UserId1 {
			user1Debt += item.Amount
		} else if item.DebtorId == settingItem.UserId2 {
			user1Debt -= item.Amount
		} else {
			log.Print("in touroku in cul debt sum")
			log.Print(settingItem.UserId1)
			log.Print(settingItem.UserId2)
			log.Fatal(item.DebtorId)
		}
		//historiesText += fmt.Sprintf("%20s\n", item.Comment)
	}
	ReplyMessageStr := ""
	if user1Debt > 0 {
		ReplyMessageStr += fmt.Sprintf("登録完了\n%sさんが%d円借りています", strings.TrimSpace(settingItem.UserName1), user1Debt)
	} else if user1Debt < 0 {
		ReplyMessageStr += fmt.Sprintf("登録完了\n%sさんが%d円借りています", strings.TrimSpace(settingItem.UserName2), user1Debt*-1)
	} else {
		ReplyMessageStr += fmt.Sprintf("登録完了\n素晴らしいことに借金はありません!")
	}
	reqStruct.Messages = []msgtask.Message{{Type: "text", Text: ReplyMessageStr}}
}

func RegisterTemplate(db *dynamodb.DynamoDB, ID string, qs []string, reqStruct *msgtask.Response) {
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
		reqStruct.Messages = []msgtask.Message{{Type: "text", Text: errMessage}}
		//reply registered messages
		err := msgtask.ReplyMessage(reqStruct)
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	//validate query cnt
	if settingItem.QueryCnt > 5 {
		errMessage := "クエリ登録の上限です\n"
		reqStruct.Messages = []msgtask.Message{{Type: "text", Text: errMessage}}
		//reply registered messages
		err = msgtask.ReplyMessage(reqStruct)
		if err != nil {
			log.Fatal(err)
		}
		return
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
	reqStruct.Messages = []msgtask.Message{{Type: "text", Text: "success"}}
	err = msgtask.ReplyMessage(reqStruct)
	if err != nil {
		log.Fatal(err)
	}
}
func GetQueryHistory() {
}
func ExecuteTempmlate(db *dynamodb.DynamoDB, ID string, qs []string, reqStruct *msgtask.Response, e msgtask.Hook_event) {
	//get room setting
	settingItem := dbtask.RoomSetting{}
	err := dbtask.GetRoomSetting(db, ID, &settingItem)
	if err != nil {
		log.Fatal(err)
	}
	//validate querycnt
	if settingItem.QueryCnt == 0 {
		errMessage := "登録されたクエリが0件です\n"
		reqStruct.Messages = []msgtask.Message{{Type: "text", Text: errMessage}}
		//reply registered messages
		err = msgtask.ReplyMessage(reqStruct)
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	registeredItem := dbtask.TemplateQuery{}
	err = dbtask.GetTemplateQuery(db, ID, qs[1], &registeredItem)
	if err != nil {
		log.Fatal(err)
	}
	if registeredItem.DebtorId == "" {
		reply := fmt.Sprintln("クエリが正しくありません")
		registeredItems := dbtask.TemplateQueries{}
		err = dbtask.GetAllTemplateQuery(db, ID, &registeredItems)
		if err != nil {
			log.Fatal(err)
		}
		reply += fmt.Sprintf("登録されたクエリは次の%d件です\n", settingItem.QueryCnt)
		for _, v := range registeredItems.Item {
			var debtorName string
			if v.DebtorId == settingItem.UserId1 {
				debtorName = settingItem.UserName1
			} else if v.DebtorId == settingItem.UserId2 {
				debtorName = settingItem.UserName2
			} else {
				reply += "内部エラー"
				break
			}
			reply += fmt.Sprintf("%s %s %d\n", v.Name, debtorName, v.Amount)
		}
		reqStruct.Messages = []msgtask.Message{{Type: "text", Text: strings.TrimRight(reply, "\n")}}
		err = msgtask.ReplyMessage(reqStruct)
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	//register registeredItem
	t := time.Now()
	m := fmt.Sprintf("%d", t.Month())
	d := fmt.Sprintf("%d", t.Day())
	if len(m) == 1 {
		m = "  " + m
	}
	if len(d) == 1 {
		d = "  " + d
	}
	date := fmt.Sprintf("%s/%s", m, d)
	err = dbtask.PutQueryHistory(db, ID, &dbtask.QueryHistory{
		RoomId:    ID,
		Timestamp: e.Timestamp,
		DebtorId:  registeredItem.DebtorId,
		Amount:    registeredItem.Amount,
		Comment:   qs[1],
		Date:      date,
		MessageId: e.Message.Id,
	})
	if err != nil {
		log.Fatal(err)
	}
	//culculate debt for notify users current state
	historyItem := dbtask.QueryHistories{}
	err = dbtask.GetQueryHistory(db, ID, &historyItem)
	if err != nil {
		log.Fatal(err)
	}
	user1Debt := int64(0)
	for _, item := range historyItem.Item {
		//culculate debt sum
		if item.DebtorId == settingItem.UserId1 {
			user1Debt += item.Amount
		} else if item.DebtorId == settingItem.UserId2 {
			user1Debt -= item.Amount
		} else {
			log.Print("in touroku in cul debt sum")
			log.Print(settingItem.UserId1)
			log.Print(settingItem.UserId2)
			log.Fatal(item.DebtorId)
		}
		//historiesText += fmt.Sprintf("%20s\n", item.Comment)
	}
	ReplyMessageStr := ""
	if user1Debt > 0 {
		ReplyMessageStr += fmt.Sprintf("登録完了\n%sさんが%d円借りています", strings.TrimSpace(settingItem.UserName1), user1Debt)
	} else if user1Debt < 0 {
		ReplyMessageStr += fmt.Sprintf("登録完了\n%sさんが%d円借りています", strings.TrimSpace(settingItem.UserName2), user1Debt*-1)
	} else {
		ReplyMessageStr += fmt.Sprintf("登録完了\n素晴らしいことに借金はありません!")
	}
	reqStruct.Messages = []msgtask.Message{{Type: "text", Text: ReplyMessageStr}}
	err = msgtask.ReplyMessage(reqStruct)
	if err != nil {
		log.Fatal(err)
	}
}
func ExecuteClearance() {
}

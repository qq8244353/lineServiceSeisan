package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/google/uuid"
	dbtask "github.com/qq8244353/lineServiceSeisan/pkg/dbtask"
	msgtask "github.com/qq8244353/lineServiceSeisan/pkg/msgtask"
	"github.com/qq8244353/lineServiceSeisan/pkg/qtask"
)

func handler(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var hevents msgtask.Hook_events
	err := json.Unmarshal([]byte(req.Body), &hevents)
	if err != nil {
		log.Fatal(err)
	}
	sess, err := session.NewSession()
	db := dynamodb.New(sess)
	if err != nil {
		log.Fatal(err)
	}
	for _, e := range hevents.Events {
		//set roomId for DB
		var ID string
		if e.Source.Type == "user" {
			ID = e.Source.UserId
		} else if e.Source.Type == "group" {
			ID = e.Source.GroupId
		} else if e.Source.Type == "room" {
			ID = e.Source.RoomId
		} else {
			log.Fatal("invalid e.Source.Type")
		}
		if e.Type == "unsend" && e.Mode == "active" {
			dbtask.UpdateDone(db, ID, false)
			//get query history
			historyItem := dbtask.QueryHistories{}
			err = dbtask.GetQueryHistory(db, ID, &historyItem)
			if err != nil {
				log.Fatal(err)
			}
			log.Print(historyItem)
			for _, item := range historyItem.Item {
				if item.MessageId == e.Unsend.MessageId {
					err = dbtask.DeleteQueryHistory(db, ID, strconv.FormatInt(item.Timestamp, 10))
					if err != nil {
						log.Fatal(err)
					}
				}
			}
			continue
		}
		//init request struct
		reqStruct := new(msgtask.Response)
		reqStruct.ReplyToken = e.ReplyToken
		//init dbtask.RoomSetting table when invited to room
		if e.Type == "join" && e.Mode == "active" {
			uuidObj1, err := uuid.NewUUID()
			if err != nil {
				log.Fatal(err)
			}
			uuidObj2, err := uuid.NewUUID()
			if err != nil {
				log.Fatal(err)
			}
			err = dbtask.PutRoomSetting(db, ID, &dbtask.RoomSetting{
				RoomId:     ID,
				UserName1:  "Tom",
				UserName2:  "Bob",
				UserId1:    uuidObj1.String(),
				UserId2:    uuidObj2.String(),
				SeisanDone: false,
				QueryCnt:   0,
			})

			if err != nil {
				log.Fatal(err)
			}
			helpText := "データーベースが正しく初期化されました\n"
			helpText += "名前はTom,Bobに初期化されています\n"
			helpText += "使用できるクエリは次の6つです\n"
			helpText += "\"init\"\n"
			helpText += "\"名前確認\"\n"
			helpText += "\"名前変更\" (変更前の名前) (変更後の名前)\n"
			helpText += "\"登録\" (借りる人の名前) (金額) (コメント)\n"
			helpText += "\"精算\"\n"
			helpText += "\"支払い完了\"\n"
			helpText += "また、登録クエリを送信取り消しした場合はそのクエリが消去されます"
			reqStruct.Messages = []msgtask.Message{{Type: "text", Text: helpText}, {Type: "sticker", PackageId: "8515", StickerId: "16581248"}}
			//reply registered messages
			err = msgtask.ReplyMessage(reqStruct)
			if err != nil {
				log.Fatal(err)
			}
			continue
		}
		if e.Type != "message" || e.Mode != "active" {
			continue
		}
		log.Print(e)
		qs := strings.Fields(e.Message.Text)
		//t = parse(e.Message.Text)
		if len(qs) == 4 && qs[0] == "クエリ登録" {
			qtask.RegisterTemplate(db, ID, qs, reqStruct)
		} else if len(qs) == 2 && qs[0] == "クエリ" {
			qtask.ExecuteTempmlate(db, ID, qs, reqStruct, e)
		} else if e.Message.Text == "クエリ確認" {
			qtask.GetTemplates(db, ID, reqStruct)
		} else if len(qs) == 3 && qs[0] == "名前変更" {
			//ok
			qtask.UpdateUsername(db, ID, qs, reqStruct)
		} else if e.Message.Text == "名前確認" {
			//get room setting
			settingItem := dbtask.RoomSetting{}
			err := dbtask.GetRoomSetting(db, ID, &settingItem)
			if err != nil {
				log.Fatal(err)
			}
			textUserName := fmt.Sprintf("ユーザー名\n%s\n%s", settingItem.UserName1, settingItem.UserName2)
			reqStruct.Messages = []msgtask.Message{{Type: "text", Text: textUserName}}
		} else if len(qs) == 4 && qs[0] == "登録" {
			qtask.RegisterQueryHistory(db, ID, qs, reqStruct, e)
		} else if e.Message.Text == "精算" {
			//get room setting
			settingItem := dbtask.RoomSetting{}
			err := dbtask.GetRoomSetting(db, ID, &settingItem)
			if err != nil {
				log.Fatal(err)
			}
			//get query history
			historyItem := dbtask.QueryHistories{}
			err = dbtask.GetQueryHistory(db, ID, &historyItem)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("%v %v", settingItem, historyItem)

			reqStruct.Messages = []msgtask.Message{}
			user1Debt := int64(0)
			var historiesText string
			for len(settingItem.UserName1) < len(settingItem.UserName2) {
				settingItem.UserName1 = "　" + settingItem.UserName1
			}
			for len(settingItem.UserName2) < len(settingItem.UserName1) {
				settingItem.UserName2 = "　" + settingItem.UserName2
			}
			sort.Slice(historyItem.Item, func(i, j int) bool { return historyItem.Item[i].Timestamp < historyItem.Item[j].Timestamp })
			for _, item := range historyItem.Item {
				if err != nil {
					log.Fatal(err)
				}
				//culculate debt sum
				userName := "undefined"
				if item.DebtorId == settingItem.UserId1 {
					user1Debt += item.Amount
					userName = settingItem.UserName1
				} else if item.DebtorId == settingItem.UserId2 {
					user1Debt -= item.Amount
					userName = settingItem.UserName2
				} else {
					log.Print(settingItem.UserId1)
					log.Print(settingItem.UserId2)
					log.Fatal(item.DebtorId)
				}
				amount := strconv.FormatInt(item.Amount, 10)
				//register history message
				if len(amount) < 5 {
					l := 5 - len(amount)
					for i := 0; i < l; i++ {
						amount = "  " + amount
					}
				}
				historiesText += fmt.Sprintf("%s %s %s %s\n", item.Date, userName, amount, item.Comment)
				//historiesText += fmt.Sprintf("%20s\n", item.Comment)
			}
			if user1Debt != 0 {
				reqStruct.Messages = append(
					reqStruct.Messages,
					msgtask.Message{Type: "text", Text: strings.TrimRight(historiesText, "\n")},
				)
			}
			//register total debt
			if user1Debt > 0 {
				reqStruct.Messages = append(
					reqStruct.Messages,
					msgtask.Message{Type: "text", Text: fmt.Sprintf("%sさんは%d円の支払いをしてください", strings.TrimSpace(settingItem.UserName1), user1Debt)},
				)
				dbtask.UpdateDone(db, ID, true)
			} else if user1Debt < 0 {
				reqStruct.Messages = append(
					reqStruct.Messages,
					msgtask.Message{Type: "text", Text: fmt.Sprintf("%sさんは%d円の支払いをしてください", strings.TrimSpace(settingItem.UserName2), user1Debt*-1)},
				)
				dbtask.UpdateDone(db, ID, true)
			} else {
				reqStruct.Messages = append(
					reqStruct.Messages,
					msgtask.Message{Type: "text", Text: "支払いはありません"},
				)
			}
		} else if e.Message.Text == "支払い完了" {
			//get room setting
			settingItem := dbtask.RoomSetting{}
			err := dbtask.GetRoomSetting(db, ID, &settingItem)
			if err != nil {
				log.Fatal(err)
			}
			if !settingItem.SeisanDone {
				reqStruct.Messages = []msgtask.Message{{Type: "text", Text: "先に精算クエリを完了してください"}}
				//reply registered messages
				err := msgtask.ReplyMessage(reqStruct)
				if err != nil {
					log.Fatal(err)
				}
				continue
			}
			historyItem := dbtask.QueryHistories{}
			err = dbtask.GetQueryHistory(db, ID, &historyItem)
			//get query history
			if err != nil {
				log.Fatal(err)
			}

			requestItemsArray := []*dynamodb.WriteRequest{}
			for _, item := range historyItem.Item {
				requestItemsArray = append(requestItemsArray, &dynamodb.WriteRequest{
					DeleteRequest: &dynamodb.DeleteRequest{
						Key: map[string]*dynamodb.AttributeValue{
							"roomId": {
								S: aws.String(item.RoomId),
							},
							"timestamp": {
								N: aws.String(strconv.FormatInt(item.Timestamp, 10)),
							},
						},
					},
				})
			}
			input := &dynamodb.BatchWriteItemInput{
				RequestItems: map[string][]*dynamodb.WriteRequest{
					"lineServiceSeisanQueryHistory": requestItemsArray,
				},
			}
			_, err = db.BatchWriteItem(input)
			if err != nil {
				log.Fatal(err)
			}
			reqStruct.Messages = []msgtask.Message{{Type: "sticker", PackageId: "8515", StickerId: "16581254"}, {Type: "text", Text: "えらいね"}}
			dbtask.UpdateDone(db, ID, false)
		} else if e.Message.Text == "デバッグ" {
			reqStruct.Messages = []msgtask.Message{
				{
					Type:    "template",
					AltText: "支払いをしてください",
					Template: msgtask.MessageTemplate{
						Type: "buttons",
						Text: "支払いをしてください",
						Actions: []msgtask.TemplateAction{
							{
								Type:  "message",
								Label: "支払い完了",
								Text:  "支払い完了",
							},
						},
					},
				},
			}
		} else if len(qs) == 2 && qs[0] == "支払日登録" {
			settingItem := dbtask.RoomSetting{}
			err := dbtask.GetRoomSetting(db, ID, &settingItem)
			if err != nil {
				log.Fatal(err)
			}
			date, err := strconv.ParseInt(qs[1], 10, 64)
			if err != nil {
				log.Fatal(err)
			}
			if date < 1 || 31 < date {
				reqStruct.Messages = []msgtask.Message{{Type: "text", Text: "支払日には1から31の整数を指定してください"}}
				//reply registered messages
				err := msgtask.ReplyMessage(reqStruct)
				if err != nil {
					log.Fatal(err)
				}
				continue
			}
			input := &dynamodb.UpdateItemInput{
				TableName: aws.String("lineServiceSeisanRoomSetting"),
				Key: map[string]*dynamodb.AttributeValue{
					"roomId": {
						S: aws.String(ID),
					},
				},
				ExpressionAttributeNames: map[string]*string{
					"#target": aws.String("paymentDue"),
				},
				ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
					":newDate": {
						N: aws.String(strconv.FormatInt(date, 10)),
					},
				},
				UpdateExpression: aws.String("set #target = :newDate"),
			}
			_, err = db.UpdateItem(input)
			if err != nil {
				log.Fatal(err)
			}
			reqStruct.Messages = []msgtask.Message{{Type: "text", Text: "success"}}
		} else if e.Message.Text == "支払日確認" {
			//get room setting
			settingItem := dbtask.RoomSetting{}
			err := dbtask.GetRoomSetting(db, ID, &settingItem)
			if err != nil {
				log.Fatal(err)
			}
			var replytext string
			if settingItem.PaymentDue < 1 || 31 < settingItem.PaymentDue {
				replytext = "支払日の設定はありません"
			} else {
				replytext = fmt.Sprintf("支払日は%d日です", settingItem.PaymentDue)
			}
			reqStruct.Messages = []msgtask.Message{{Type: "text", Text: replytext}}
		}

		//reply registered messages
		err := msgtask.ReplyMessage(reqStruct)
		if err != nil {
			log.Fatal(err)
		}
	}

	return events.APIGatewayProxyResponse{
		Body:       "",
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(handler)
}

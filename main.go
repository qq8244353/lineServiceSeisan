package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/uuid"
)

type Hooked_events_arr struct {
	Events []Hooked_events
}

type Hooked_events struct {
	Type    string
	Message struct {
		Type string
		Id   string
		Text string
	}
	Timestamp int64
	Source    struct {
		Type    string
		GroupId string
		RoomId  string
		UserId  string
	}
	ReplyToken string
	Mode       string
	/*
		WebHookEventId  string
		DeliveryContext struct {
			IsRedelivery string
		}
	*/
	Unsend struct {
		MessageId string
	}
}

type Resp struct {
	ReplyToken string    `json:"replyToken"`
	Messages   []Message `json:"messages"`
}

type Message struct {
	Type      string   `json:"type"`
	Text      string   `json:"text"`
	PackageId string   `json:"packageId"`
	StickerId string   `json:"stickerId"`
	AltText   string   `json:"altText"`
	Template  Template `json:"template"`
}

type Template struct {
	Type    string   `json:"type"`
	Text    string   `json:"text"`
	Actions []Action `json:"actions"`
}

type Action struct {
	Type  string `json:"type"`
	Label string `json:"label"`
	Text  string `json:"text"`
}

type RoomSetting struct {
	RoomId     string `dynamodbav:"roomId"`
	UserName1  string `dynamodbav:"userName1"`
	UserName2  string `dynamodbav:"userName2"`
	UserId1    string `dynamodbav:"userId1"`
	UserId2    string `dynamodbav:"userId2"`
	SeisanDone bool   `dynamodbav:"seisanDone"`
	QueryCnt   int64  `dynamodbav:"queryCnt"`
}

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

type RegisteredQueries struct {
	Item []RegisteredQuery
}

type RegisteredQuery struct {
	RoomId   string `dynamodbav:"roomId"`
	Name     string `dynamodbav:"name"`
	DebtorId string `dynamodbav:"debtorId"`
	Amount   int64  `dynamodbav:"amount"`
}

// reply registered messages
func replyMessage(reqStruct *Resp) error {
	reqJson, err := json.Marshal(&reqStruct)
	if err != nil {
		return err
	}
	log.Print(string(reqJson))
	req, err := http.NewRequest(
		"POST",
		"https://api.line.me/v2/bot/message/reply",
		bytes.NewBuffer(reqJson),
	)
	if err != nil {
		return err
	}
	accessToken := "Bearer " + os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", accessToken)
	client := new(http.Client)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	dumpResp, _ := httputil.DumpResponse(resp, true)
	log.Printf("%s", dumpResp)
	return nil
}

func getRoomSetting(db *dynamodb.DynamoDB, ID string, settingItem *RoomSetting) error {
	//get roomSetting
	getParam := &dynamodb.GetItemInput{
		TableName: aws.String("lineServiceSeisanRoomSetting"),
		Key: map[string]*dynamodb.AttributeValue{
			"roomId": {
				S: aws.String(ID),
			},
		},
	}
	log.Print(ID)
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

func getRegisteredQuery(db *dynamodb.DynamoDB, ID string, name string, registeredItem *RegisteredQuery) error {
	//get registeredQuery
	getParam := &dynamodb.GetItemInput{
		TableName: aws.String("lineServiceSeisanRegisteredQuery"),
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

// get query history
func getQueryHistory(db *dynamodb.DynamoDB, ID string, historyItem *QueryHistories) error {
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

// get registered all query
func getAllRegisteredQuery(db *dynamodb.DynamoDB, ID string, registeredItem *RegisteredQueries) error {
	getParamQuery := &dynamodb.QueryInput{
		TableName:              aws.String("lineServiceSeisanRegisteredQuery"),
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
		p := RegisteredQuery{}
		err = dynamodbattribute.UnmarshalMap(v, &p)
		registeredItem.Item = append(registeredItem.Item, p)
	}
	if err != nil {
		return err
	}
	return err
}
func updateDone(db *dynamodb.DynamoDB, roomId string, b bool) {
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
		log.Fatal(err)
	}
}

func handler(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var hevents Hooked_events_arr
	err := json.Unmarshal([]byte(req.Body), &hevents)
	if err != nil {
		log.Fatal(err)
	}
	log.Print(hevents)
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
			updateDone(db, ID, false)
			//get query history
			historyItem := QueryHistories{}
			err = getQueryHistory(db, ID, &historyItem)
			if err != nil {
				log.Fatal(err)
			}
			log.Print(historyItem)
			for _, item := range historyItem.Item {
				if item.MessageId == e.Unsend.MessageId {
					input := &dynamodb.DeleteItemInput{
						TableName: aws.String("lineServiceSeisanQueryHistory"),
						Key: map[string]*dynamodb.AttributeValue{
							"roomId": {
								S: aws.String(ID),
							},
							"timestamp": {
								N: aws.String(strconv.FormatInt(item.Timestamp, 10)),
							},
						},
						ReturnConsumedCapacity:      aws.String("NONE"),
						ReturnItemCollectionMetrics: aws.String("NONE"),
						ReturnValues:                aws.String("NONE"),
					}
					_, err = db.DeleteItem(input)
					if err != nil {
						log.Fatal(err)
					}
				}
			}
			continue
		}
		//init request struct
		reqStruct := new(Resp)
		reqStruct.ReplyToken = e.ReplyToken
		//init roomSetting table when invited to room
		if e.Type == "join" && e.Mode == "active" {
			uuidObj1, err := uuid.NewUUID()
			if err != nil {
				log.Fatal(err)
			}
			uuidObj2, err := uuid.NewUUID()
			if err != nil {
				log.Fatal(err)
			}
			inputAV, err := dynamodbattribute.MarshalMap(RoomSetting{
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
			input := &dynamodb.PutItemInput{
				TableName: aws.String("lineServiceSeisanRoomSetting"),
				Item:      inputAV,
			}
			_, err = db.PutItem(input)
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
			reqStruct.Messages = []Message{{Type: "text", Text: helpText}, {Type: "sticker", PackageId: "8515", StickerId: "16581248"}}
			//reply registered messages
			err = replyMessage(reqStruct)
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
			//get room setting
			settingItem := RoomSetting{}
			err := getRoomSetting(db, ID, &settingItem)
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
			inputAV, err := dynamodbattribute.MarshalMap(RegisteredQuery{
				RoomId:   ID,
				Name:     qs[1],
				DebtorId: debtorId,
				Amount:   amount,
			})
			if err != nil {
				log.Fatal(err)
			}
			inputPut := &dynamodb.PutItemInput{
				TableName: aws.String("lineServiceSeisanRegisteredQuery"),
				Item:      inputAV,
			}
			_, err = db.PutItem(inputPut)
			if err != nil {
				log.Fatal(err)
			}
			//reply registered messages
			reqStruct.Messages = []Message{{Type: "text", Text: "success"}}
			err = replyMessage(reqStruct)
			if err != nil {
				log.Fatal(err)
			}
		} else if len(qs) == 2 && qs[0] == "クエリ" {
			//get room setting
			settingItem := RoomSetting{}
			err := getRoomSetting(db, ID, &settingItem)
			if err != nil {
				log.Fatal(err)
			}
			//validate querycnt
			if settingItem.QueryCnt == 0 {
				errMessage := "登録されたクエリが0件です\n"
				reqStruct.Messages = []Message{{Type: "text", Text: errMessage}}
				//reply registered messages
				err = replyMessage(reqStruct)
				if err != nil {
					log.Fatal(err)
				}
				continue
			}
			registeredItem := RegisteredQuery{}
			err = getRegisteredQuery(db, ID, qs[1], &registeredItem)
			if err != nil {
				log.Fatal(err)
			}
			if registeredItem.DebtorId == "" {
				reply := fmt.Sprintln("クエリが正しくありません")
				registeredItems := RegisteredQueries{}
				err = getAllRegisteredQuery(db, ID, &registeredItems)
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
				reqStruct.Messages = []Message{{Type: "text", Text: strings.TrimRight(reply, "\n")}}
				err = replyMessage(reqStruct)
				if err != nil {
					log.Fatal(err)
				}
				continue
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
			inputAV, err := dynamodbattribute.MarshalMap(QueryHistory{
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
			input := &dynamodb.PutItemInput{
				TableName: aws.String("lineServiceSeisanQueryHistory"),
				Item:      inputAV,
			}
			_, err = db.PutItem(input)
			if err != nil {
				log.Fatal(err)
			}
			//culculate debt for notify users current state
			historyItem := QueryHistories{}
			err = getQueryHistory(db, ID, &historyItem)
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
			replyMessageStr := ""
			if user1Debt > 0 {
				replyMessageStr += fmt.Sprintf("登録完了\n%sさんが%d円借りています", strings.TrimSpace(settingItem.UserName1), user1Debt)
			} else if user1Debt < 0 {
				replyMessageStr += fmt.Sprintf("登録完了\n%sさんが%d円借りています", strings.TrimSpace(settingItem.UserName2), user1Debt*-1)
			} else {
				replyMessageStr += fmt.Sprintf("登録完了\n素晴らしいことに借金はありません!")
			}
			reqStruct.Messages = []Message{{Type: "text", Text: replyMessageStr}}
			err = replyMessage(reqStruct)
			if err != nil {
				log.Fatal(err)
			}
		} else if e.Message.Text == "クエリ確認" {
			//get room setting
			settingItem := RoomSetting{}
			err := getRoomSetting(db, ID, &settingItem)
			if err != nil {
				log.Fatal(err)
			}
			registeredItem := RegisteredQueries{}
			err = getAllRegisteredQuery(db, ID, &registeredItem)
			if err != nil {
				log.Fatal(err)
			}
			reply := fmt.Sprintf("登録されたクエリは次の%d件です\n", settingItem.QueryCnt)
			for _, v := range registeredItem.Item {
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
			reqStruct.Messages = []Message{{Type: "text", Text: strings.TrimRight(reply, "\n")}}
			err = replyMessage(reqStruct)
			if err != nil {
				log.Fatal(err)
			}
		} else if e.Message.Text == "init" {
			uuidObj1, err := uuid.NewUUID()
			if err != nil {
				log.Fatal(err)
			}
			uuidObj2, err := uuid.NewUUID()
			if err != nil {
				log.Fatal(err)
			}
			inputAV, err := dynamodbattribute.MarshalMap(RoomSetting{
				RoomId:    ID,
				UserName1: "Tom",
				UserName2: "Bob",
				UserId1:   uuidObj1.String(),
				UserId2:   uuidObj2.String(),
				QueryCnt:  0,
			})
			if err != nil {
				log.Fatal(err)
			}
			input := &dynamodb.PutItemInput{
				TableName: aws.String("lineServiceSeisanRoomSetting"),
				Item:      inputAV,
			}
			_, err = db.PutItem(input)
			if err != nil {
				log.Fatal(err)
			}
		} else if len(qs) == 3 && qs[0] == "名前変更" {
			//get room setting
			settingItem := RoomSetting{}
			err := getRoomSetting(db, ID, &settingItem)
			if err != nil {
				log.Fatal(err)
			}
			log.Print(qs)
			log.Printf("%d %d %d", len(qs[0]), len(qs[1]), len(qs[2]))
			if len(qs[2]) > 15 {
				reqStruct.Messages = []Message{{Type: "text", Text: "ユーザー名は5文字以下にしてください"}}
				//reply registered messages
				err := replyMessage(reqStruct)
				if err != nil {
					log.Fatal(err)
				}
				continue
			}
			if (qs[1] == settingItem.UserName1 && qs[2] == settingItem.UserName2) || (qs[1] == settingItem.UserName2 && qs[2] == settingItem.UserName1) {
				errMessage := fmt.Sprintf("ユーザーが重複しています\n%s\n%s", settingItem.UserName1, settingItem.UserName2)
				reqStruct.Messages = []Message{{Type: "text", Text: errMessage}}
				//reply registered messages
				err := replyMessage(reqStruct)
				if err != nil {
					log.Fatal(err)
				}
				continue
			}
			var toBeReplaced string
			if qs[1] == settingItem.UserName1 {
				toBeReplaced = "userName1"
			} else if qs[1] == settingItem.UserName2 {
				toBeReplaced = "userName2"
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
			reqStruct.Messages = []Message{{Type: "text", Text: "success"}}
		} else if len(qs) == 4 && qs[0] == "登録" {
			updateDone(db, ID, false)
			//get room setting
			settingItem := RoomSetting{}
			err := getRoomSetting(db, ID, &settingItem)
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
				reqStruct.Messages = []Message{{Type: "text", Text: errMessage}}
				//reply registered messages
				err := replyMessage(reqStruct)
				if err != nil {
					log.Fatal(err)
				}
				continue
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
			inputAV, err := dynamodbattribute.MarshalMap(QueryHistory{
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
			input := &dynamodb.PutItemInput{
				TableName: aws.String("lineServiceSeisanQueryHistory"),
				Item:      inputAV,
			}
			_, err = db.PutItem(input)
			if err != nil {
				log.Fatal(err)
			}
			// reqStruct.Messages = []Message{{Type: "text", Text: "success"}}
			//culculate debt for notify users current state
			historyItem := QueryHistories{}
			err = getQueryHistory(db, ID, &historyItem)
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
			replyMessageStr := ""
			if user1Debt > 0 {
				replyMessageStr += fmt.Sprintf("登録完了\n%sさんが%d円借りています", strings.TrimSpace(settingItem.UserName1), user1Debt)
			} else if user1Debt < 0 {
				replyMessageStr += fmt.Sprintf("登録完了\n%sさんが%d円借りています", strings.TrimSpace(settingItem.UserName2), user1Debt*-1)
			} else {
				replyMessageStr += fmt.Sprintf("登録完了\n素晴らしいことに借金はありません!")
			}
			reqStruct.Messages = []Message{{Type: "text", Text: replyMessageStr}}
		} else if e.Message.Text == "精算" {
			//get room setting
			settingItem := RoomSetting{}
			err := getRoomSetting(db, ID, &settingItem)
			if err != nil {
				log.Fatal(err)
			}
			log.Print(settingItem)
			//get query history
			historyItem := QueryHistories{}
			err = getQueryHistory(db, ID, &historyItem)
			if err != nil {
				log.Fatal(err)
			}
			log.Print(historyItem)

			reqStruct.Messages = []Message{}
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
					Message{Type: "text", Text: strings.TrimRight(historiesText, "\n")},
				)
			}
			//register total debt
			if user1Debt > 0 {
				reqStruct.Messages = append(
					reqStruct.Messages,
					Message{Type: "text", Text: strings.TrimSpace(settingItem.UserName1) + " " + strconv.FormatInt(user1Debt, 10)},
				)
				reqStruct.Messages = append(
					reqStruct.Messages,
					Message{Type: "text", Text: "支払いをしてください"},
				)
				updateDone(db, ID, true)
			} else if user1Debt < 0 {
				reqStruct.Messages = append(
					reqStruct.Messages,
					Message{Type: "text", Text: strings.TrimSpace(settingItem.UserName2) + " " + strconv.FormatInt(user1Debt*-1, 10)},
				)
				// unconveinent feature
				// reqStruct.Messages = append(reqStruct.Messages,
				// 	Message{
				// 		Type:    "template",
				// 		AltText: "支払いをしてください",
				// 		Template: Template{
				// 			Type: "buttons",
				// 			Text: "支払いをしてください",
				// 			Actions: []Action{
				// 				{
				// 					Type:  "message",
				// 					Label: "支払い完了",
				// 					Text:  "支払い完了",
				// 				},
				// 			},
				// 		},
				// 	},
				// )
				reqStruct.Messages = append(
					reqStruct.Messages,
					Message{Type: "text", Text: "支払いをしてください"},
				)
				updateDone(db, ID, true)
			} else {
				reqStruct.Messages = append(
					reqStruct.Messages,
					Message{Type: "text", Text: "支払いはありません"},
				)
			}
		} else if e.Message.Text == "支払い完了" {
			//get room setting
			settingItem := RoomSetting{}
			err := getRoomSetting(db, ID, &settingItem)
			if err != nil {
				log.Fatal(err)
			}
			if !settingItem.SeisanDone {
				reqStruct.Messages = []Message{{Type: "text", Text: "先に精算クエリを完了してください"}}
				//reply registered messages
				err := replyMessage(reqStruct)
				if err != nil {
					log.Fatal(err)
				}
				continue
			}
			historyItem := QueryHistories{}
			err = getQueryHistory(db, ID, &historyItem)
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
			reqStruct.Messages = []Message{{Type: "sticker", PackageId: "8515", StickerId: "16581254"}, {Type: "text", Text: "えらいね"}}
			updateDone(db, ID, false)
		} else if e.Message.Text == "名前確認" {
			//get room setting
			settingItem := RoomSetting{}
			err := getRoomSetting(db, ID, &settingItem)
			if err != nil {
				log.Fatal(err)
			}
			textUserName := fmt.Sprintf("ユーザー名\n%s\n%s", settingItem.UserName1, settingItem.UserName2)
			reqStruct.Messages = []Message{{Type: "text", Text: textUserName}}
		} else if e.Message.Text == "デバッグ" {
			reqStruct.Messages = []Message{
				{
					Type:    "template",
					AltText: "支払いをしてください",
					Template: Template{
						Type: "buttons",
						Text: "支払いをしてください",
						Actions: []Action{
							{
								Type:  "message",
								Label: "支払い完了",
								Text:  "支払い完了",
							},
						},
					},
				},
			}
		} else {
			//ignore if message is unexpected format
			// helpText := "クエリを正しく処理できませんでした\n"
			// helpText += fmt.Sprint(qs) + "\n"
			// helpText += "使用できるクエリは次の6つです\n"
			// helpText += "\"init\"\n"
			// helpText += "\"名前確認\"\n"
			// helpText += "\"名前変更\" (変更前の名前) (変更後の名前)\n"
			// helpText += "\"登録\" (借りる人の名前) (金額) (コメント)\n"
			// helpText += "\"精算\"\n"
			// helpText += "\"支払い完了\"\n"
			// helpText += "また、登録クエリを送信取り消しした場合はそのクエリが消去されます"
			// reqStruct.Messages = []Message{{Type: "text", Text: helpText}, {Type: "sticker", PackageId: "8515", StickerId: "16581259"}}
		}

		//reply registered messages
		err := replyMessage(reqStruct)
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

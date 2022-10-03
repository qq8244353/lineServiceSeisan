package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
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
}

type Resp struct {
	ReplyToken string    `json:"replyToken"`
	Messages   []Message `json:"messages"`
}

type Message struct {
	Type      string `json:"type"`
	Text      string `json:"text"`
	PackageId string `json:"packageId"`
	StickerId string `json:"stickerId"`
}

type RoomSetting struct {
	RoomId    string `dynamodbav:"roomId"`
	UserName1 string `dynamodbav:"userName1"`
	UserName2 string `dynamodbav:"userName2"`
	UserId1   string `dynamodbav:"userId1"`
	UserId2   string `dynamodbav:"userId2"`
}

type QueryHistory struct {
	RoomId    string `dynamodbav:"roomId"`
	Timestamp int64  `dynamodbav:"timestamp"`
	Comment   string `dynamodbav:"comment"`
	Date      string `dynamodbav:"date"`
	DebtorId  string `dynamodbav:"debtorId"`
	Amount    int64  `dynamodbav:"amount"`
}

type QueryHistories struct {
	Item []QueryHistory
}

// reply registered messages
func replyMessage(reqStruct *Resp) error {
	reqJson, err := json.Marshal(&reqStruct)
	if err != nil {
		return err
	}
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
func getRoomSetting(db *dynamodb.DynamoDB, e *Hooked_events, settingItem *RoomSetting) error {
	//get roomSetting
	getParam := &dynamodb.GetItemInput{
		TableName: aws.String("lineServiceSeisanRoomSetting"),
		Key: map[string]*dynamodb.AttributeValue{
			"roomId": {
				S: aws.String(e.Source.UserId),
			},
		},
	}
	log.Print(e.Source.UserId)
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

// get query history
func getQueryHistory(db *dynamodb.DynamoDB, e *Hooked_events, historyItem *QueryHistories) error {
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
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":roomId": {
				S: aws.String(e.Source.UserId),
			},
		},
		ProjectionExpression: aws.String("#roomId, #timestamp, #comment, #date, #amount, #debtorId"),
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
		if e.Type == "join" {

		}
		if e.Type != "message" || e.Mode != "active" {
			continue
		}
		log.Print(e)
		reqStruct := new(Resp)
		reqStruct.ReplyToken = e.ReplyToken
		qs := strings.Fields(e.Message.Text)
		//t = parse(e.Message.Text)
		if e.Message.Text == "init" {
		} else if len(qs) == 3 && qs[0] == "名前変更" {
			//get room setting
			settingItem := RoomSetting{}
			err := getRoomSetting(db, &e, &settingItem)
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
				errMessage := fmt.Sprintf("ユーザー名が重複しています\n%s\n%s", settingItem.UserName1, settingItem.UserName2)
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
						S: aws.String(e.Source.UserId),
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
			//get room setting
			settingItem := RoomSetting{}
			err := getRoomSetting(db, &e, &settingItem)
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
			date := fmt.Sprintf("%d/%d", t.Month(), t.Day())
			inputAV, err := dynamodbattribute.MarshalMap(QueryHistory{
				RoomId:    e.Source.UserId,
				Timestamp: e.Timestamp,
				DebtorId:  debtorId,
				Amount:    amount,
				Comment:   qs[3],
				Date:      date,
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
			reqStruct.Messages = []Message{{Type: "text", Text: "success"}}
		} else if e.Message.Text == "精算" {
			//get query history
			historyItem := QueryHistories{}
			err = getQueryHistory(db, &e, &historyItem)
			if err != nil {
				log.Fatal(err)
			}
			log.Print(historyItem)
			//get room setting
			settingItem := RoomSetting{}
			err := getRoomSetting(db, &e, &settingItem)
			if err != nil {
				log.Fatal(err)
			}
			log.Print(settingItem)

			reqStruct.Messages = []Message{}
			user1Debt := int64(0)
			var historiesText string
			for len(settingItem.UserName1) < len(settingItem.UserName2) {
				settingItem.UserName1 = "　" + settingItem.UserName1
			}
			for len(settingItem.UserName2) < len(settingItem.UserName1) {
				settingItem.UserName2 = "　" + settingItem.UserName2
			}
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
				if len(item.Date) < 5 {
					l := 5 - len(item.Date)
					for i := 0; i < l; i++ {
						item.Date = "  " + item.Date
					}
				}
				//register history message
				historiesText += fmt.Sprintf("%s %s %6s %s\n", item.Date, userName, strconv.FormatInt(item.Amount, 10), item.Comment)
				//historiesText += fmt.Sprintf("%20s\n", item.Comment)
			}
			reqStruct.Messages = append(
				reqStruct.Messages,
				Message{Type: "text", Text: historiesText},
			)
			//register total debt
			if user1Debt > 0 {
				reqStruct.Messages = append(
					reqStruct.Messages,
					Message{Type: "text", Text: settingItem.UserName1 + " " + strconv.FormatInt(user1Debt, 10)},
				)
				reqStruct.Messages = append(reqStruct.Messages, Message{Type: "text", Text: "支払いをしてください"})
			} else if user1Debt < 0 {
				reqStruct.Messages = append(
					reqStruct.Messages,
					Message{Type: "text", Text: settingItem.UserName2 + " " + strconv.FormatInt(user1Debt*-1, 10)},
				)
				reqStruct.Messages = append(reqStruct.Messages, Message{Type: "text", Text: "支払いをしてください"})
			} else {
				reqStruct.Messages = append(
					reqStruct.Messages,
					Message{Type: "text", Text: "支払いはありません"},
				)
			}
		} else if e.Message.Text == "支払い完了" {
			//get room setting
			settingItem := RoomSetting{}
			err := getRoomSetting(db, &e, &settingItem)
			if err != nil {
				log.Fatal(err)
			}
			historyItem := QueryHistories{}
			err = getQueryHistory(db, &e, &historyItem)
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
			reqStruct.Messages = []Message{{Type: "text", Text: "えらいね"}}
		} else if e.Message.Text == "uuid" {
			uuidObj, _ := uuid.NewUUID()
			reqStruct.Messages = []Message{{Type: "text", Text: uuidObj.String()}}
		} else if e.Message.Text == "timestamp" {
			reqStruct.Messages = []Message{{Type: "sticker", PackageId: "8515", StickerId: "16581254"}}
		} else if e.Message.Text == "デバッグ" {
			db := dynamodb.New(sess)
			//get roomSetting
			getParam := &dynamodb.GetItemInput{
				TableName: aws.String("lineServiceSeisanRoomSetting"),
				Key: map[string]*dynamodb.AttributeValue{
					"roomId": {
						S: aws.String(e.Source.UserId),
					},
				},
			}
			dbRes, err := db.GetItem(getParam)
			if err != nil {
				log.Fatal(err)
			}
			item := RoomSetting{}
			err = dynamodbattribute.UnmarshalMap(dbRes.Item, &item)
			if err != nil {
				log.Fatal(err)
			}
			reqStruct.Messages = []Message{{Type: "text", Text: "ユーザー名" + item.UserName1 + item.UserName2}}
		} else if e.Message.Text == "デバッグ登録" {
			debugReg(e)
		} else {
			reqStruct.Messages = []Message{{Type: "text", Text: "クエリの意味が理解できません"}}
			reqStruct.Messages = []Message{{Type: "text", Text: fmt.Sprint(qs)}}
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

func debugReg(e Hooked_events) {
	sess, err := session.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	db := dynamodb.New(sess)
	//get roomSetting
	getParam := &dynamodb.GetItemInput{
		TableName: aws.String("lineServiceSeisanRoomSetting"),
		Key: map[string]*dynamodb.AttributeValue{
			"roomId": {
				S: aws.String(e.Source.UserId),
			},
		},
	}
	log.Print(e.Source.UserId)
	dbRes, err := db.GetItem(getParam)
	if err != nil {
		log.Fatal(err)
	}
	item := RoomSetting{}
	err = dynamodbattribute.UnmarshalMap(dbRes.Item, &item)
	if err != nil {
		log.Fatal(err)
	}
	vec := []QueryHistory{
		{RoomId: e.Source.UserId, Timestamp: int64(1664425009743), DebtorId: item.UserId2, Comment: "No4モーニング", Amount: int64(1620), Date: "10/23"},
		{RoomId: e.Source.UserId, Timestamp: int64(1664425183370), DebtorId: item.UserId1, Comment: "イタリアン", Amount: int64(3377), Date: "9/23"},
		{RoomId: e.Source.UserId, Timestamp: int64(1664425344172), DebtorId: item.UserId1, Comment: "餃子", Amount: int64(1130), Date: "9/25"},
	}
	log.Print(vec)
	for _, qh := range vec {
		log.Print(qh)
		inputAV, err := dynamodbattribute.MarshalMap(qh)
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
	}
}

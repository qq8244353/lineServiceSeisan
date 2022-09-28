package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
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
	/*
		Timestamp string `json:"timestamp"`
	*/
	Source struct {
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
	Type string `json:"type"`
	Text string `json:"text"`
}

func handler(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var hevents Hooked_events_arr
	err := json.Unmarshal([]byte(req.Body), &hevents)
	if err != nil {
		log.Fatal(err)
	}
	log.Print(hevents)
	for _, e := range hevents.Events {
		if e.Type != "message" || e.Mode != "active" {
			continue
		}
		log.Print(e)
		reqStruct := new(Resp)
		reqStruct.ReplyToken = e.ReplyToken
		//t = parse(e.Message.Text)
		if e.Message.Text == "ユーザー登録" {
			reqStruct.Messages = []Message{{Type: "text", Text: "あ"}}
		} else if e.Message.Text == "名前変更" {
			reqStruct.Messages = []Message{{Type: "text", Text: "い"}}
		} else if e.Message.Text == "登録" {
			reqStruct.Messages = []Message{{Type: "text", Text: "う"}}
		} else if e.Message.Text == "精算" {
			reqStruct.Messages = []Message{{Type: "text", Text: "支払いをしてください"}}
		} else if e.Message.Text == "支払い完了" {
			reqStruct.Messages = []Message{{Type: "text", Text: "えらいね"}}
		} else {
			reqStruct.Messages = []Message{{Type: "text", Text: "クエリの意味が理解できません"}}
		}
		reqJson, err := json.Marshal(reqStruct)
		if err != nil {
			log.Fatal(err)
		}

		log.Print(string(reqJson))

		req, err := http.NewRequest(
			"POST",
			"https://api.line.me/v2/bot/message/reply",
			bytes.NewBuffer(reqJson),
		)
		if err != nil {
			log.Fatal(err)
		}
		accessToken := "Bearer " + os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", accessToken)
		client := new(http.Client)
		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		dumpResp, _ := httputil.DumpResponse(resp, true)
		log.Printf("%s", dumpResp)
	}

	return events.APIGatewayProxyResponse{
		Body:       "",
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(handler)
}

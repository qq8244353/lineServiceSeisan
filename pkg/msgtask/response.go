package msgtask

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
)

type Response struct {
	ReplyToken string    `json:"replyToken"`
	Messages   []Message `json:"messages"`
	To         string    `json:"to"`
}

type Message struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	PackageId string          `json:"packageId"`
	StickerId string          `json:"stickerId"`
	AltText   string          `json:"altText"`
	Template  MessageTemplate `json:"template"`
}

type MessageTemplate struct {
	Type    string           `json:"type"`
	Text    string           `json:"text"`
	Actions []TemplateAction `json:"actions"`
}

type TemplateAction struct {
	Type  string `json:"type"`
	Label string `json:"label"`
	Text  string `json:"text"`
}

// reply registered messages
func ReplyMessage(reqStruct *Response) error {
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

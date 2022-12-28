package msgtask

type Hook_events struct {
	Events []Hook_event
}

type Hook_event struct {
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
	// WebHookEventId  string
	// DeliveryContext struct {
	// 	IsRedelivery string
	// }
	Unsend struct {
		MessageId string
	}
}

package utils

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"time"

	"github.com/dsc-bot/webhook-service/config"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Message struct {
	Webhook struct {
		ID    string  `json:"id"`
		URL   string  `json:"url"`
		Token *string `json:"token"`
	} `json:"webhook"`
	Data struct {
		Type        string `json:"type"`
		Bot         string `json:"bot"`
		User        string `json:"user"`
		BotId       string `json:"bot_id"`
		UserId      string `json:"user_id"`
		ListingId   string `json:"listing_id"`
		WebhookName string `json:"webhook_name"`
		Test        bool   `json:"test"`
		Weight      int    `json:"weight"`
		Query       string `json:"query"`
	} `json:"data"`
	Version int   `json:"version"`
	Count   int   `json:"count"`
	SentAt  int64 `json:"sentAt"`
}

func (msg *Message) DeprecatedFields() {
	// Support older webhooks
	if msg.Version == 0 {
		if msg.Data.Bot == "" && msg.Data.BotId != "" {
			msg.Data.Bot = msg.Data.BotId
		}
		if msg.Data.User == "" && msg.Data.UserId != "" {
			msg.Data.User = msg.Data.UserId
		}
	}
}

func ProcessMessage(ch *amqp.Channel, msg amqp.Delivery) {
	var parsedMsg Message
	if err := json.Unmarshal(msg.Body, &parsedMsg); err != nil {
		Logger.Sugar().Errorf("Failed to parse message: %v", err)
		return
	}

	parsedMsg.DeprecatedFields()
	count := parsedMsg.Count
	sentAt := parsedMsg.SentAt
	now := time.Now().Unix()

	if count >= 2 {
		delay := int(math.Pow(2, float64(count)))
		if now-sentAt < int64(delay) {
			Logger.Sugar().Debugf("Requeueing message: now (%d) - sentAt (%d) < %d seconds", now, sentAt, delay)
			requeueMessage(ch, parsedMsg)
			return
		}
	}

	if !sendWebhook(parsedMsg) && parsedMsg.Count < 10 {
		parsedMsg.Count++
		requeueMessage(ch, parsedMsg)
	}
}

func sendWebhook(msg Message) bool {
	webhookURL := msg.Webhook.URL
	token := msg.Webhook.Token
	data, _ := json.Marshal(msg.Data)

	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(data))
	if err != nil {
		Logger.Sugar().Warnf("Failed to create request for %s: %v", webhookURL, err)
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "DscBot-Webhook/1.0 (+https://dsc.bot)")
	if token != nil {
		req.Header.Set("Authorization", *token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		Logger.Sugar().Warnf("HTTP request to %s, failed: %v", webhookURL, err)
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		Logger.Sugar().Infof("Sent webhook to %s, Response: %d (Server error), requeuing message", webhookURL, resp.StatusCode)
		return false
	}

	Logger.Sugar().Infof("Sent webhook to %s, Response: %d", webhookURL, resp.StatusCode)
	return true
}

func requeueMessage(ch *amqp.Channel, msg Message) {
	newMsgBody, _ := json.Marshal(msg)
	err := ch.Publish("", config.Conf.Rabbit.Queue, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        newMsgBody,
	})
	if err != nil {
		Logger.Sugar().Errorf("Failed to requeue message: %v", err)
	}
}

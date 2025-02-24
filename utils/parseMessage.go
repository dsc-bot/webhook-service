package utils

import (
	"bytes"
	"encoding/json"
	"log"
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
		BotID       string `json:"bot_id"`
		UserID      string `json:"user_id"`
		WebhookName string `json:"webhook_name"`
		Test        bool   `json:"test"`
		Weight      int    `json:"weight"`
		Query       string `json:"query"`
	} `json:"data"`
	Count  int   `json:"count,omitempty"`
	SentAt int64 `json:"sentAt,omitempty"`
}

func ProcessMessage(ch *amqp.Channel, msg amqp.Delivery) {
	var parsedMsg Message
	if err := json.Unmarshal(msg.Body, &parsedMsg); err != nil {
		Logger.Sugar().Errorf("Failed to parse message: %v", err)
		return
	}

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
		log.Printf("Failed to create request: %v", err)
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	if token != nil {
		req.Header.Set("Authorization", *token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		Logger.Sugar().Debugf("HTTP request failed: %v", err)
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		Logger.Sugar().Debugf("Server error (%d), requeuing message", resp.StatusCode)
		return false
	}

	Logger.Sugar().Debugf("Sent webhook to %s, Response: %d", webhookURL, resp.StatusCode)
	return true
}

func requeueMessage(ch *amqp.Channel, msg Message) {
	newMsgBody, _ := json.Marshal(msg)
	err := ch.Publish("", config.Conf.Rabbit.Queue, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        newMsgBody,
	})
	if err != nil {
		log.Printf("Failed to requeue message: %v", err)
	}
}

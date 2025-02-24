package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/dsc-bot/webhook-service/config"
	"github.com/dsc-bot/webhook-service/utils"
	amqp "github.com/rabbitmq/amqp091-go"
)

var shutdown = false
var shutdownMutex sync.Mutex

func main() {
	config.Parse()
	lerr := utils.Configure(nil, config.Conf.JsonLogs, config.Conf.LogLevel)
	failOnError(lerr, "Failed to create zap logger")

	table := amqp.NewConnectionProperties()
	table.SetClientConnectionName("webhook-service")

	conn, err := amqp.DialConfig(config.Conf.Rabbit.Url, amqp.Config{
		Properties: table,
	})
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(config.Conf.Rabbit.Queue, true, false, false, false, nil)
	failOnError(err, "Failed to declare a queue")

	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	failOnError(err, "Failed to register a consumer")

	var wg sync.WaitGroup

	go func() {
		for d := range msgs {
			shutdownMutex.Lock()
			if shutdown {
				shutdownMutex.Unlock()
				utils.Logger.Info("Shutdown in progress, messages requeued and exiting")
				d.Nack(false, true)
				break
			}
			shutdownMutex.Unlock()

			wg.Add(1)
			go func(delivery amqp.Delivery) {
				defer wg.Done()

				// Process the message
				utils.ProcessMessage(ch, delivery)
				delivery.Ack(false)
			}(d)
		}
	}()

	utils.Logger.Info("Waiting for messages...")

	// keep alive until shutdown signal
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)
	<-shutdownCh

	utils.Logger.Info("Received shutdown signal, stopping processing...")

	shutdownMutex.Lock()
	shutdown = true // Set shutdown flag
	shutdownMutex.Unlock()

	wg.Wait() // Wait for all in-flight messages to finish

	utils.Logger.Info("Shutdown complete")
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

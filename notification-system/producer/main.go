package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	rmq "github.com/rabbitmq/rabbitmq-amqp-go-client/pkg/rabbitmqamqp"
)

func main() {
	log.SetFlags(log.Lshortfile)

	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatalln("failed to load .env")
	}

	RMQ_URI := os.Getenv("RMQ_URI")

	////////////////////////////////////////////////////////////////
	ctx := context.Background()
	env := rmq.NewEnvironment(RMQ_URI, nil)

	conn, err := env.NewConnection(ctx)
	if err != nil {
		log.Fatalln("failed to open connection:", err)
	}
	defer env.CloseConnections(context.Background())

	// create a notifications exchange
	exchangeInfo, err := conn.Management().DeclareExchange(ctx, &rmq.TopicExchangeSpecification{
		Name: "notifications",
	})
	if err != nil {
		log.Fatalln("failed to declare an exchange:", err)
	}

	// create a publisher
	for i := range 10 {
		notification, routingKey := randomNotification(i)

		publisher, err := conn.NewPublisher(ctx, &rmq.ExchangeAddress{
			Exchange: exchangeInfo.Name(),
			Key:      routingKey,
		}, nil)
		if err != nil {
			log.Fatalln("failed to create publisher:", err)
		}

		// publish messages
		payload, _ := json.Marshal(notification)
		msg := rmq.NewMessage(payload)

		res, err := publisher.Publish(ctx, msg)
		if err != nil {
			log.Fatalln("failed to publish message:", err)
		}

		switch res.Outcome.(type) {
		case *rmq.StateAccepted:
		case *rmq.StateRejected:
			log.Fatalln("message rejected:", res.Outcome)
		case *rmq.StateReleased:
			log.Fatalln("message released:", res.Outcome)
		case *rmq.StateModified:
			log.Fatalln("message modified:", res.Outcome)
		default:
			log.Fatalln("unknown publish outcome:", res.Outcome)
		}
		publisher.Close(context.Background())
		time.Sleep(300 * time.Millisecond)
	}
}

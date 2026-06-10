package main

import (
	"context"
	"errors"
	"log"

	rmq "github.com/rabbitmq/rabbitmq-amqp-go-client/pkg/rabbitmqamqp"
)

const brokerURI = "amqp://guest:guest@localhost:5672"

func main() {
	ctx := context.Background()

	env := rmq.NewEnvironment(brokerURI, nil)

	conn, err := env.NewConnection(ctx)
	if err != nil {
		log.Panicln("failed to connect to rmq:", err)
	}

	defer func() {
		_ = env.CloseConnections(context.Background())
	}()

	qInfo, err := conn.Management().DeclareQueue(ctx, &rmq.QuorumQueueSpecification{Name: "hello"})
	if err != nil {
		log.Panicln("failed to declare a queue:", err)
	}

	consumer, err := conn.NewConsumer(ctx, qInfo.Name(), nil)
	if err != nil {
		log.Panicln("failed to create a new consumer:", err)
	}

	defer func() {
		_ = consumer.Close(context.Background())
	}()

	log.Println("[*] waiting for messages...")

	for {
		delivery, err := consumer.Receive(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			log.Panicln("failed to receive a message:", err)
		}

		msg := delivery.Message()
		var body string

		if len(msg.Data) > 0 {
			body = string(msg.Data[0])
		}
		log.Println("received a message:", body)
		// settle(Accept, Discard, Requeue) only after the work is done
		// if a consumer dies without settling, rmq can re-deliver to another consumer
		// making sure that no message is lost
		err = delivery.Accept(ctx)
		if err != nil {
			log.Panicln("failed to accept message:", err)
		}
	}
}

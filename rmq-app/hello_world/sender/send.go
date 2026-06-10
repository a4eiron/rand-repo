package main

import (
	"context"
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

	publisher, err := conn.NewPublisher(ctx, &rmq.QueueAddress{Queue: qInfo.Name()}, nil)
	if err != nil {
		log.Panicln("failed to create publisher:", err)
	}
	defer func() {
		publisher.Close(context.Background())
	}()

	body := "Hello world"
	res, err := publisher.Publish(ctx, rmq.NewMessage([]byte(body)))
	if err != nil {
		log.Panicln("failed to publish a message:", err)
	}

	switch res.Outcome.(type) {
	case *rmq.StateAccepted:
	default:
		log.Panicln("unexpected publish outcome:", res.Outcome)
	}

	log.Println("[x] sent:", body)
}

package main

import (
	"context"
	"log"
	"os"
	"strings"

	rmq "github.com/rabbitmq/rabbitmq-amqp-go-client/pkg/rabbitmqamqp"
)

const borkerURL = "amqp://guest:guest@localhost:5672"

func main() {
	ctx := context.Background()
	env := rmq.NewEnvironment(borkerURL, nil)

	conn, err := env.NewConnection(ctx)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = env.CloseConnections(context.Background())
	}()

	qInfo, err := conn.Management().DeclareQueue(ctx, &rmq.QuorumQueueSpecification{Name: "task_queue"})
	if err != nil {
		panic(err)
	}

	publisher, err := conn.NewPublisher(ctx, &rmq.QueueAddress{Queue: qInfo.Name()}, nil)
	if err != nil {
		panic(err)
	}
	defer func() {
		publisher.Close(context.Background())
	}()

	body := bodyFrom(os.Args)
	res, err := publisher.Publish(ctx, rmq.NewMessage([]byte(body)))
	if err != nil {
		panic(err)
	}

	switch res.Outcome.(type) {
	case *rmq.StateAccepted:
	default:
		log.Panicln("unexpected publish outcome:", res.Outcome)
	}

	log.Println("[x] sent:", body)
}

func bodyFrom(args []string) string {
	var s string
	if (len(args) < 2) || args[1] == "" {
		s = "hello"
	} else {
		s = strings.Join(args[1:], " ")
	}

	return s
}

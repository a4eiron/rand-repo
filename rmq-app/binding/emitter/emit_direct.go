package main

import (
	"context"
	"log"
	"os"
	"strings"

	rmq "github.com/rabbitmq/rabbitmq-amqp-go-client/pkg/rabbitmqamqp"
)

const brokerURI = "amqp://guest:guest@localhost:5672"

func main() {
	ctx := context.Background()
	env := rmq.NewEnvironment(brokerURI, nil)

	conn, err := env.NewConnection(ctx)
	if err != nil {
		panic(err)
	}
	defer func() {
		env.CloseConnections(context.Background())
	}()

	exInfo, err := conn.Management().DeclareExchange(ctx, &rmq.DirectExchangeSpecification{Name: "logs_direct"})
	if err != nil {
		panic(err)
	}

	publisher, err := conn.NewPublisher(ctx, &rmq.ExchangeAddress{Exchange: exInfo.Name(), Key: severityFrom(os.Args)}, nil)
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
	if (len(args) < 3) || args[2] == "" {
		s = "hello"
	} else {
		s = strings.Join(args[2:], " ")
	}
	return s
}

func severityFrom(args []string) string {
	var s string
	if (len(args) < 2) || args[1] == "" {
		s = "info"
	} else {
		s = args[1]
	}
	return s
}

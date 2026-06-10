package main

import (
	"context"
	"errors"
	"log"
	"math/rand/v2"
	"time"

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
		env.CloseConnections(context.Background())
	}()

	qInfo, err := conn.Management().DeclareQueue(ctx, &rmq.QuorumQueueSpecification{
		Name: "task_queue",
	})
	if err != nil {
		panic(err)
	}

	consumer, err := conn.NewConsumer(ctx, qInfo.Name(), &rmq.ConsumerOptions{
		InitialCredits: 1, // limits the messages inflight (analogous to prefetch)
	})
	if err != nil {
		panic(err)
	}

	defer func() {
		consumer.Close(context.Background())
	}()

	log.Println("[*] waiting for messages...")
	for {
		delivery, err := consumer.Receive(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			log.Println("failed to recieve message:", err)
		}

		msg := delivery.Message()
		var payload []byte

		if len(msg.Data) > 0 {
			payload = msg.Data[0]
		}

		log.Println("received a message:", string(payload))

		time.Sleep(time.Duration(rand.IntN(3)) * time.Second)

		log.Println("done")
		err = delivery.Accept(ctx)
		if err != nil {
			log.Panicln("failed to accept message:", err)
		}

	}
}

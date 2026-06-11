package main

import (
	"fmt"
	"math/rand/v2"
)

type NotificationType string

const (
	Email NotificationType = "email"
	SMS   NotificationType = "sms"
)

type Priority string

const (
	Trasactional Priority = "transactional"
	Marketing    Priority = "marketing"
)

type Notification struct {
	ID       string           `json:"id"`
	Type     NotificationType `json:"type"`
	Priority Priority         `json:"priority"`
	To       string           `json:"to"`
	Body     string           `json:"body"`
}

func randomNotification(i int) (Notification, string) {
	types := []NotificationType{Email, SMS}
	priorities := []Priority{Trasactional, Marketing}

	t := types[rand.IntN(len(types))]
	p := priorities[rand.IntN(len(priorities))]

	routingKey := fmt.Sprintf("notifications.%s", t)

	n := Notification{
		ID:       fmt.Sprintf("notif-%d", i),
		Type:     t,
		Priority: p,
		To:       fmt.Sprintf("use%d@a4corp.com", i),
		Body:     "hello!, hi!, hola!",
	}

	return n, routingKey
}

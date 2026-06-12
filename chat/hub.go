package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var ServerID = uuid.New().String()

type Message struct {
	UserID   string `json:"user_id"`
	RoomID   string `json:"room_id"`
	Data     []byte `json:"data"`
	DataStr  string `json:"data_str"`
	ServerID string `json:"server_id"`
}

type Client struct {
	ID     string
	roomID string
	conn   *websocket.Conn
	send   chan []byte
}

func (c *Client) ReadPump(ctx context.Context, hub *Hub) {
	defer func() {
		hub.unregister <- c
	}()

	for {
		_, msg, err := c.conn.Read(ctx)
		if err != nil {
			return
		}
		hub.broadcast <- Message{UserID: c.ID, RoomID: c.roomID, Data: msg, ServerID: ServerID}
	}
}

func (c *Client) WritePump(ctx context.Context, hub *Hub) {
	defer func() {
		hub.unregister <- c
	}()

	pinger := time.NewTicker(20 * time.Second)
	defer pinger.Stop()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			writeCtx, writeCancel := context.WithTimeout(ctx, 5*time.Second)
			err := c.conn.Write(writeCtx, websocket.MessageText, msg)
			writeCancel()
			if err != nil {
				return
			}
		case <-pinger.C:
			pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
			err := c.conn.Ping(pingCtx)
			pingCancel()
			if err != nil {
				log.Println(err)
				return
			}
		}
	}
}

type Hub struct {
	rooms      map[string]map[*Client]bool
	rdb        *redis.Client
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	hub := &Hub{
		rooms:      make(map[string]map[*Client]bool),
		broadcast:  make(chan Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}

	hub.rdb = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	return hub
}

func (h *Hub) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			for room := range h.rooms {
				for client := range h.rooms[room] {
					client.conn.CloseNow()
				}
			}
			return

		case c := <-h.register:
			if h.rooms[c.roomID] == nil {
				h.rooms[c.roomID] = make(map[*Client]bool)
			}
			h.rooms[c.roomID][c] = true

		case c := <-h.unregister:
			if room, ok := h.rooms[c.roomID]; ok {
				if _, ok := room[c]; ok {
					log.Println("BYE", c.roomID, c.ID)
					delete(h.rooms[c.roomID], c)
					close(c.send)
					c.conn.CloseNow()
				}
			}

		case msg := <-h.broadcast:
			data, err := json.Marshal(msg)
			if err != nil {
				log.Println(err)
				continue
			}
			h.broadCastToRoom(msg.RoomID, msg.UserID, data)
			h.publishToRedis(msg.RoomID, data)

		}
	}
}

func (h *Hub) publishToRedis(roomID string, msg []byte) error {
	return h.rdb.Publish(context.Background(), fmt.Sprintf("chat:room:%s", roomID), msg).Err()
}

func (h *Hub) subscribeToRedis(ctx context.Context) {
	pubsub := h.rdb.PSubscribe(ctx, "chat:room:*")
	ch := pubsub.Channel()

	for {
		select {
		case <-ctx.Done():
			return
		case redisMsg, ok := <-ch:
			if !ok {
				return
			}

			var msg Message
			err := json.Unmarshal([]byte(redisMsg.Payload), &msg)
			if err != nil {
				log.Println("unmarshal", err)
				continue
			}

			if msg.ServerID != ServerID {
				h.broadCastToRoom(msg.RoomID, msg.UserID, []byte(redisMsg.Payload))
			}
		}
	}
}

func (h *Hub) broadCastToRoom(roomID string, userID string, msg []byte) {
	for client := range h.rooms[roomID] {
		if client.ID == userID {
			continue
		}
		select {
		case client.send <- msg:
		default:
			h.unregister <- client
		}
	}
}

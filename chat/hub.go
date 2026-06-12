package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/coder/websocket"
)

type Message struct {
	UserID string `json:"user_id"`
	RoomID string `json:"room_id"`
	Data   []byte `json:"data"`
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
		hub.broadcast <- Message{UserID: c.ID, RoomID: c.roomID, Data: msg}
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
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]map[*Client]bool),
		broadcast:  make(chan Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
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

			for c := range h.rooms[msg.RoomID] {
				select {
				case c.send <- data:
				default:
					// delete(h.rooms[c.roomID], c)
					h.unregister <- c
				}
			}

		}
	}
}

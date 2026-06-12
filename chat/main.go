package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coder/websocket"
)

func main() {
	log.SetFlags(log.Lshortfile)

	if len(os.Args) < 2 {
		log.Fatalln("want port")
	}

	hub := NewHub()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	svr := &http.Server{
		Addr:    ":" + os.Args[1],
		Handler: http.HandlerFunc(handleWS(hub)),
	}

	go func() {
		log.Println("server listening on port:", os.Args[1])
		if err := svr.ListenAndServe(); err != nil {
			log.Fatalln(err)
		}
	}()

	go hub.run(ctx)
	go hub.subscribeToRedis(ctx)

	<-ctx.Done()
	log.Println("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	svr.Shutdown(shutdownCtx)
}

func handleWS(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roomID := r.Header.Get("X-Room-Id")
		userID := r.Header.Get("X-User-Id")
		if roomID == "" || userID == "" {
			http.Error(w, "X-Room-Id or X-User-Id header missing", http.StatusBadRequest)
			return
		}
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
		if err != nil {
			log.Println(err)
			return
		}

		log.Println(roomID)
		log.Println(userID)

		client := &Client{
			ID:     userID,
			roomID: roomID,
			conn:   conn,
			send:   make(chan []byte, 64),
		}
		hub.register <- client

		// ctx := r.Context()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go client.WritePump(ctx, hub)

		client.ReadPump(ctx, hub)
	}
}

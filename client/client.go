package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

type message struct {
	data string
}

type Client struct {
	conn *websocket.Conn
	id   int32
}

func (c *Client) Send(msg []byte) {
	err := c.conn.WriteMessage(websocket.TextMessage, msg)
	if err != nil {
		log.Println("write:", err)
		return
	}
}

func (c *Client) Recv(f func([]byte)) {
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}
		f(message)
		// log.Printf("recv: %s", message)
		// f(string(message))
	}
}
func (c *Client) init() {
	flag.Parse()
	log.SetFlags(0)

	// interrupt := make(chan os.Signal, 1)
	// signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/echo"}
	log.Printf("connecting to %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	c.conn = conn
	_, message, err := c.conn.ReadMessage()
	if err != nil {
		log.Println("read:", err)
		return
	}
	fmt.Sscanf(string(message), "%d", &c.id)
	println(c.id)

	// defer c.Close()

	// done := make(chan struct{})

	// go func() {
	// 	defer close(done)
	// 	for {
	// 		_, message, err := c.ReadMessage()
	// 		if err != nil {
	// 			log.Println("read:", err)
	// 			return
	// 		}
	// 		log.Printf("recv: %s", message)
	// 	}
	// }()

	// ticker := time.NewTicker(time.Second)
	// defer ticker.Stop()

	// for {
	// 	select {
	// 	case <-done:
	// 		return
	// 	case t := <-ticker.C:
	// 		err := c.WriteMessage(websocket.TextMessage, []byte(t.String()))
	// 		if err != nil {
	// 			log.Println("write:", err)
	// 			return
	// 		}
	// 	case <-interrupt:
	// 		log.Println("interrupt")

	// 		// Cleanly close the connection by sending a close message and then
	// 		// waiting (with timeout) for the server to close the connection.
	// 		err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	// 		if err != nil {
	// 			log.Println("write close:", err)
	// 			return
	// 		}
	// 		select {
	// 		case <-done:
	// 		case <-time.After(time.Second):
	// 		}
	// 		return
	// 	}
	// }
}

package main

import (
	"fmt"
	"github.com/tv2169145/grpc-chat/chat-proto"
	"google.golang.org/grpc"
	"io"
	"net"
	"sync"
)
type Connection struct {
	conn chat.Chat_ChatServer
	send chan *chat.ChatMessage
	quit chan struct{}
}

func NewConnection(conn chat.Chat_ChatServer) *Connection {
	newConnection := &Connection{
		conn: conn,
		send: make(chan *chat.ChatMessage),
		quit: make(chan struct{}),
	}
	go newConnection.start()
	return newConnection
}

func (c *Connection) start() {
	running := true
	for running {
		select {
		case msg := <-c.send:
			c.conn.Send(msg)
		case <-c.quit:
			running = false
		}
	}
}

func (c *Connection) Send(msg *chat.ChatMessage) {
	defer func() {
		recover()
	}()
	c.send <- msg
}

func (c *Connection) GetMessage(broadcast chan<- *chat.ChatMessage) error {
	for {
		msg, err := c.conn.Recv()
		if err == io.EOF {
			c.Close()
			return nil
		} else if err != nil {
			c.Close()
			return err
		}
		go func(msg *chat.ChatMessage) {
			select {
			case broadcast <- msg:
			case <-c.quit:
			}
		}(msg)
	}
}

func (c *Connection) Close() error {
	close(c.send)
	close(c.quit)
	return nil
}



// 封裝多個連線, 建立連線池---------
type ChatServer struct {
	broadcast chan *chat.ChatMessage
	quit chan struct{}
	connections []*Connection
	connLock sync.Mutex
}

// 初始化連線池, 並開始監聽各連線
func NewChatServer() *ChatServer {
	srv := &ChatServer{
		broadcast: make(chan *chat.ChatMessage),
		quit: make(chan struct{}),
	}
	go srv.start()
	return srv
}

func (s *ChatServer) Close() error {
	close(s.quit)
	return nil
}

func (s *ChatServer) start() {
	running := true
	for running {
		select {
		case msg := <-s.broadcast:
			s.connLock.Lock()
			for _, singleConnection := range s.connections {
				go singleConnection.Send(msg)
			}
			s.connLock.Unlock()
		case <-s.quit:
			running = false
		}
	}
}

func (s *ChatServer) Chat(stream chat.Chat_ChatServer) error {
	conn := NewConnection(stream)
	s.connLock.Lock()
	s.connections = append(s.connections, conn)
	s.connLock.Unlock()
	err := conn.GetMessage(s.broadcast)

	s.connLock.Lock()
	for i, v := range s.connections {
		if v == conn {
			s.connections = append(s.connections[:i], s.connections[i+1:]...)
		}
	}
	s.connLock.Unlock()
	return err
}

func main () {
	lst, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}
	s := grpc.NewServer()
	srv := NewChatServer()
	chat.RegisterChatServer(s, srv)
	fmt.Println("Now serving at port 8080")
	err = s.Serve(lst)
	if err != nil {
		panic(err)
	}
}

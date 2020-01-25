package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/tv2169145/grpc-chat/chat-proto"
	"google.golang.org/grpc"
	"io"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("請輸入正確參數格式 (127.0.0.1:8080 jimmy)")
		return
	}
	ctx := context.Background()
	conn, err := grpc.Dial(os.Args[1], grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	c := chat.NewChatClient(conn)
	stream, err := c.Chat(ctx)

	if err != nil {
		panic(err)
	}
	waitc := make(chan struct{})

	go func() {
		for {
			msg, err := stream.Recv()
			if err == io.EOF {
				close(waitc)
				return
			} else if err != nil {
				panic(err)
			}
			fmt.Println(msg.User + ":" + msg.Message)
		}
	}()

	fmt.Println("Connection established, type \"quit\" or use ctrl+c to exit")
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		msg := scanner.Text()
		if msg == "quit" || msg == "exit" {
			err := stream.CloseSend()
			if err != nil {
				panic(err)
			}
			break
		}
		err := stream.Send(&chat.ChatMessage{
			User: os.Args[2],
			Message: msg,
		})
		if err != nil {
			panic(err)
		}
	}
	<-waitc
}

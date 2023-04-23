package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	hellopb "grpctutorial/pkg/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	scanner *bufio.Scanner
	client  hellopb.GreetingServiceClient
)

func main() {
	//スタート時間・処理時間表示
	startTime := time.Now()
	fmt.Printf("client start\ttime: %v \n", startTime)
	defer func() {
		fmt.Printf("\n processing time: %v", time.Since(startTime).Milliseconds())
	}()

	//標準入力から文字列を受け取るスキャナを用意
	scanner = bufio.NewScanner(os.Stdin)

	//gRPCserverとのコネクションを確率
	address := "localhost:8080"
	conn, err := grpc.Dial(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("connection failed.")
		return
	}

	//gRPCクライアントを作成
	client = hellopb.NewGreetingServiceClient(conn)

	for {
		fmt.Println("-1: exit")
		fmt.Println("1: send Request")
		fmt.Println("2: Server Stream")
		fmt.Println("3: Client Stream")
		fmt.Printf("please enter >>")

		scanner.Scan()
		in := scanner.Text()

		switch in {
		case "-1":
			fmt.Println("bye.")
			//TODO gotoを消す
			goto M

		case "1":
			hello()

		case "2":
			HelloServerStream()

		case "3":
			HelloClientStream()

		}
	}
M:
}

// Unary RPCがリクエストを送るところ
func hello() {
	fmt.Println("Pleace enter your name")
	scanner.Scan()
	name := scanner.Text()

	req := &hellopb.HelloRequest{
		Name: name,
	}

	// Helloメソッドの実行 -> HelloResponse型のレスポンスresを入手
	res, err := client.Hello(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(res.GetMessage())
	}
}

// serverストリームがリクエストを複数送るところ
func HelloServerStream() {
	fmt.Println("Plase enter your name.")
	scanner.Scan()
	name := scanner.Text()
	req := &hellopb.HelloRequest{
		Name: name,
	}
	//サーバーから複数回レスポンスを受け取るためのストリームを得る
	stream, err := client.HelloServerStream(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	}

	for {
		//ストリームからレスポンスを得る
		res, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			fmt.Println("all the responses have already received.")
			break
		}

		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(res)
	}

}

// Client Stream RPCがリクエストを送るところ
func HelloClientStream() {
	// サーバーに複数回リクエストを送るためのストリームを得る
	stream, err := client.HelloClientStream(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}

	sendCount := 5
	fmt.Printf("Please enter %v names.\n", sendCount)
	// サーバーに送るリクエストを全て送信
	for i := 0; i < sendCount; i++ {
		scanner.Scan()
		name := scanner.Text()

		// ストリームを通じてリクエストを送信
		if err := stream.Send(&hellopb.HelloRequest{
			Name: name,
		}); err != nil {
			fmt.Println(err)
			return
		}
	}

	//ストリームからレスポンスを得る
	res, err := stream.CloseAndRecv()
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(res.GetMessage())
	}

}

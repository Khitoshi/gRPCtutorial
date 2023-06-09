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

	Interceptors "grpctutorial/cmd/client/Interceptor"
	hellopb "grpctutorial/pkg/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
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
		grpc.WithUnaryInterceptor(Interceptors.MyUnaryClientInteceptor1),
		grpc.WithStreamInterceptor(Interceptors.MyStreamClientInteceptor1),
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
		fmt.Println("4: Bi Stream")
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

		case "4":
			HelloBiStream()
		}
	}
M:
}

// Unary RPCがリクエストを送るところ
func hello() {
	fmt.Println("Pleace enter your name")
	//入力
	scanner.Scan()
	name := scanner.Text()

	//serverのUnaryRPCを呼び出し
	req := &hellopb.HelloRequest{
		Name: name,
	}

	//メタデータ
	ctx := context.Background()
	md := metadata.New(map[string]string{"type": "unary", "from": "client"})
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Helloメソッドの実行 -> HelloResponse型のレスポンスresを入手
	var header, trailer metadata.MD
	res, err := client.Hello(ctx, req, grpc.Header(&header), grpc.Trailer(&trailer))
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(header)
		fmt.Println(trailer)
		fmt.Println(res.GetMessage())
	}
}

// serverストリームがリクエストを複数送るところ
func HelloServerStream() {
	fmt.Println("Plase enter your name.")
	//入力
	scanner.Scan()
	name := scanner.Text()
	//serverのClientStreamRPCを呼び出し
	req := &hellopb.HelloRequest{
		Name: name,
	}
	//サーバーから複数回レスポンスを受け取るためのストリームを得る
	stream, err := client.HelloServerStream(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	}
	sendDone := make(chan bool)
	//受信
	go func() {
		defer close(sendDone)
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
	}()

	//スレッド終了まで待機
	<-sendDone
}

// Client Stream RPCがリクエストを送るところ
func HelloClientStream() {
	//serverのClientStreamRPCと接続
	stream, err := client.HelloClientStream(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}

	sendDone := make(chan bool)

	go func() {
		defer close(sendDone)
		//送信回数
		sendCount := 5
		fmt.Printf("Please enter %d names.\n", sendCount)
		for i := 0; i < sendCount; i++ {
			//入力
			scanner.Scan()
			name := scanner.Text()

			//送信
			if err := stream.Send(&hellopb.HelloRequest{
				Name: name,
			}); err != nil {
				fmt.Println(err)
				return
			}
		}

	}()

	//スレッド終了まで待機
	<-sendDone

	//受信
	res, err := stream.CloseAndRecv()
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(res.GetMessage())
	}
}

// 双方性streaming
func HelloBiStream() {
	//メタデータ
	ctx := context.Background()
	// 新しいメタデータを作成し、キーと値のペアを設定します
	md := metadata.New(map[string]string{"type": "stream", "from": "client"})
	//ctxに格納
	ctx = metadata.NewOutgoingContext(ctx, md)

	//serverの双方向ストリーミングRPCメソッドと接続
	stream, err := client.HelloBiStreams(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}

	//送信回数
	sendNum := 5
	fmt.Printf("Please enter %d names.\n", sendNum)

	//送信カウント
	sendCount := 0

	//送信時チャンネル
	sendDone := make(chan bool)
	//送信時チャンネル
	recvDone := make(chan bool)

	//送信処理
	go func() {
		defer close(sendDone)
		for {
			//入力
			scanner.Scan()
			name := scanner.Text()
			sendCount++
			//送信
			if err := stream.Send(&hellopb.HelloRequest{
				Name: name,
			}); err != nil {
				fmt.Println(err)
				break
			}
			//sendNum回行うと終了する
			if sendCount == sendNum {
				if err := stream.CloseSend(); err != nil {
					fmt.Println(err)
				}
				break
			}

		}
	}()

	//受信処理
	go func() {
		defer close(recvDone)

		for {
			var headerMD metadata.MD
			//ヘッダー情報が存在しない時
			if headerMD == nil {
				headerMD, err = stream.Header()
				if err != nil {
					fmt.Println(err)
				} else {
					fmt.Println(headerMD)
				}
			}

			//受信
			if res, err := stream.Recv(); err != nil {
				if !errors.Is(err, io.EOF) {
					//error内容を表示
					fmt.Println(err)
				}
				break
			} else {
				//受信内容を表示
				fmt.Println(res.GetMessage())
			}

		}
	}()

	//トレーラー出力
	trailerMD := stream.Trailer()
	fmt.Println(trailerMD)

	//チャンネルが両方閉じるまで待機
	<-sendDone
	<-recvDone
}

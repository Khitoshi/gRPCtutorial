package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	Interceptors "grpctutorial/cmd/server/Interceptor"
	hellopb "grpctutorial/pkg/grpc"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"

	"google.golang.org/grpc"
)

type myServer struct {
	hellopb.UnimplementedGreetingServiceServer
}

// Unary RPCがレスポンスを返すところ
func (m *myServer) Hello(ctx context.Context, req *hellopb.HelloRequest) (*hellopb.HelloResponse, error) {
	//ctxからメタデータを取得
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		log.Println(md)
	}

	//ヘッダーを作成
	headerMD := metadata.New(map[string]string{"type": "unary", "from": "server", "in": "header"})
	if err := grpc.SetHeader(ctx, headerMD); err != nil {
		return nil, err
	}

	//トレイラーを作成
	trailerMD := metadata.New(map[string]string{"type": "unary", "from": "server", "in": "trailer"})
	if err := grpc.SetTrailer(ctx, trailerMD); err != nil {
		return nil, err
	}

	// HelloResponse型を1つreturnする
	// (Unaryなので、レスポンスを一つ返せば終わり)
	return &hellopb.HelloResponse{
		Message: fmt.Sprintf("hello %s", req.GetName()),
	}, nil
}

// Server Stream RPCがレスポンスを返すところ
func (s *myServer) HelloServerStream(req *hellopb.HelloRequest, stream hellopb.GreetingService_HelloServerStreamServer) error {
	//serverが送信する回数
	resCound := 5
	for i := 0; i < resCound; i++ {
		// streamのSendメソッドを使っている
		if err := stream.Send(&hellopb.HelloResponse{
			//reqに送信されたデータが入っている
			Message: fmt.Sprintf("[%d] Hello, %s!", i, req.GetName()),
		}); err != nil {
			return err
		}
		//1秒 待機
		time.Sleep(time.Second * 1)
	}
	return nil
}

// Client Stream RPCがリクエストを受け取るところ
func (s *myServer) HelloClientStream(stream hellopb.GreetingService_HelloClientStreamServer) error {
	//受信した名前追加
	nameList := make([]string, 0)
	for {
		//streamのRecvメソッドを呼び出してリクエスト内容を取得する
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			//リクエストを全て受け取ったので纏めて返す!
			message := fmt.Sprintf("Hello ,%v!", nameList)
			//送信して閉じる
			return stream.SendAndClose(&hellopb.HelloResponse{
				Message: message,
			})
		}
		if err != nil {
			return err
		}
		//名前のリストに新しい名前追加
		nameList = append(nameList, req.GetName())
	}
}

func (s *myServer) HelloBiStreams(stream hellopb.GreetingService_HelloBiStreamsServer) error {
	//ストリームのコンテキストからメタデータを取得
	if md, ok := metadata.FromIncomingContext(stream.Context()); ok {
		log.Println(md)
	}

	//すぐにヘッダーを送信
	headerMD := metadata.New(map[string]string{"type": "stream", "from": "server", "in": "header"})
	if err := stream.SendHeader(headerMD); err != nil {
		return err
	}

	//トレイラー作成
	trailerMD := metadata.New(map[string]string{"type": "stream", "from": "server", "in": "trailer"})
	stream.SetTrailer(trailerMD)

	errChan := make(chan error, 1)
	go func() {
		for {
			req, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				errChan <- nil
				return
			}
			if err != nil {
				errChan <- err
				return
			}
			message := fmt.Sprintf("Hello, %v!", req.GetName())
			if err := stream.Send(&hellopb.HelloResponse{
				Message: message,
			}); err != nil {
				errChan <- nil
				return
			}
		}
	}()
	return <-errChan
}

// 自作サービス構造体のコンストラクタを定義
func NewMyServer() *myServer {
	return &myServer{}
}

func main() {
	//ポート番号8080のLisnterを作成
	port := 8080
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(err)
	}

	//gRPCserverを作成
	s := grpc.NewServer(
		grpc.UnaryInterceptor(Interceptors.MyUnaryServerInterceptor1),
		grpc.StreamInterceptor(Interceptors.MyStreamServerInterceptor1),
	)

	//gRPCサーバーにGreetingServiceを登録
	hellopb.RegisterGreetingServiceServer(s, NewMyServer())

	//serverリフレクションの設定
	reflection.Register(s)

	//作成したgRPCserverを稼働させる
	go func() {
		log.Printf("start gRPC server port: %v", port)
		s.Serve(listener)
	}()

	//Ctrl+Cが入力されたらGraceful shutdownされるようにする
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("stopping gRPC server...")
	s.GracefulStop()
}

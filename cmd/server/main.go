package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	hellopb "grpctutorial/pkg/grpc"

	"google.golang.org/grpc/reflection"

	"google.golang.org/grpc"
)

type myServer struct {
	hellopb.UnimplementedGreetingServiceServer
}

// Unary RPCがレスポンスを返すところ
func (m *myServer) Hello(ctx context.Context, req *hellopb.HelloRequest) (*hellopb.HelloResponse, error) {

	// HelloResponse型を1つreturnする
	// (Unaryなので、レスポンスを一つ返せば終わり)
	return &hellopb.HelloResponse{
		Message: fmt.Sprintf("hello %s", req.GetName()),
	}, nil
}

// Server Stream RPCがレスポンスを返すところ
func (s *myServer) HelloServerStream(req *hellopb.HelloRequest, stream hellopb.GreetingService_HelloServerStreamServer) error {
	resCound := 5
	for i := 0; i < resCound; i++ {
		// streamのSendメソッドを使っている
		if err := stream.Send(&hellopb.HelloResponse{
			Message: fmt.Sprintf("[%d] Hello, %s!", i, req.GetName()),
		}); err != nil {
			return err
		}
		//1秒 待機
		time.Sleep(time.Second * 1)
	}
	return nil
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
	s := grpc.NewServer()

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

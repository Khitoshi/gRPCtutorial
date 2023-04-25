package Interceptors

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
)

func MyUnaryClientInteceptor1(ctx context.Context, method string, req, res interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

	// リクエスト送信前に割り込ませる前処理
	fmt.Println("[pre] my unary client interceptor 1", method, req)

	// 本来のリクエスト
	err := invoker(ctx, method, req, res, cc, opts...)

	// リクエスト送信後に割り込ませる後処理
	fmt.Println("[post] my unary client interceptor 1", res)
	return err
}

package server

import (
    "golang.org/x/net/context"
    "net"
    "google.golang.org/grpc/grpclog"
    "google.golang.org/grpc"
    pb "gocron/modules/rpc/proto"
    "gocron/modules/utils"
    "gocron/modules/rpc/auth"
    "google.golang.org/grpc/credentials"
)

type Server struct {}

func (s Server) Run(ctx context.Context, req *pb.TaskRequest) (*pb.TaskResponse, error)  {
    defer func() {
        if err := recover(); err != nil {
            grpclog.Println(err)
        }
    } ()
    output, err := utils.ExecShell(ctx, req.Command)
    resp := new(pb.TaskResponse)
    resp.Output = output
    if err != nil {
        resp.Error = err.Error()
    } else {
        resp.Error = ""
    }

    return resp, nil
}

func Start(addr string, enableTLS bool, certificate auth.Certificate)  {
    defer func() {
       if err := recover(); err != nil {
           grpclog.Println("panic", err)
       }
    } ()

    l, err := net.Listen("tcp", addr)
    if err != nil {
        grpclog.Fatal(err)
    }

    var s *grpc.Server
    if enableTLS {
        tlsConfig, err := certificate.GetTLSConfigForServer()
        if err != nil {
            grpclog.Fatal(err)
        }
        opt := grpc.Creds(credentials.NewTLS(tlsConfig))
        s = grpc.NewServer(opt)
        pb.RegisterTaskServer(s, Server{})
        grpclog.Printf("listen %s with TLS", addr)
    } else {
        s = grpc.NewServer()
        pb.RegisterTaskServer(s, Server{})
        grpclog.Printf("listen %s", addr)
    }

    err = s.Serve(l)
    grpclog.Fatal(err)
}


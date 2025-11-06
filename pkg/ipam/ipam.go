package ipam

import (
	"context"
	"log"
	"net"
	"os"

	pb "github.com/Frank-svg-dev/ppnb-cni/rpc"
	"google.golang.org/grpc"
)

const PPNBSocketPath = "/var/run/ppnb.sock"

type server struct {
	pb.UnimplementedIPAMServer
	//ipPool map[string]string
}

func (s *server) AllocateIP(ctx context.Context, req *pb.AllocateIPRequest) (*pb.AllocateIPResponse, error) {
	podIP, err := allocateIP(req.ContainerId)
	if err != nil {
		return nil, err
	}

	return &pb.AllocateIPResponse{
		Ip: podIP,
		//Gateway: "10.0.0.1",
		//Subnet:  "10.0.0.0/24",
	}, nil
}

func (s *server) ReleaseIP(ctx context.Context, req *pb.ReleaseIPRequest) (*pb.ReleaseIPResponse, error) {
	err := releaseIP(req.ContainerId)
	if err != nil {
		return nil, err
	}
	return &pb.ReleaseIPResponse{Success: true, Message: "ip released"}, nil
}

func StartIPAMServer() error {
	if _, err := os.Stat(PPNBSocketPath); err == nil {
		err := os.Remove(PPNBSocketPath)
		if err != nil {
			log.Println("创建socket失败")
			return err
		}
	}

	lis, err := net.Listen("unix", PPNBSocketPath)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	defer lis.Close()

	s := grpc.NewServer()
	pb.RegisterIPAMServer(s, &server{})

	log.Printf("gRPC IPAM server running on unix socket %s", PPNBSocketPath)
	if err := s.Serve(lis); err != nil {
		log.Println("failed to serve: %v", err)
		return err
	}

	return nil
}

package cni

import (
	"context"
	"log"
	"time"

	pb "github.com/Frank-svg-dev/ppnb-cni/rpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
)

func IpApplicationFunc(containerID string) (string, string, error) {
	conn, err := grpc.Dial(
		"unix://"+PPNBSocketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  50 * time.Millisecond,
				Multiplier: 1.5,
				MaxDelay:   1 * time.Second,
			},
			MinConnectTimeout: 3 * time.Second,
		}),
	)
	if err != nil {
		log.Println("did not connect: %v\n", err)
		return "", "", err
	}
	defer conn.Close()

	client := pb.NewIPAMClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	resp, err := client.AllocateIP(ctx, &pb.AllocateIPRequest{
		ContainerId: containerID,
	})
	if err != nil {
		log.Println("AllocateIP failed: %v", err)
		return "", "", err
	}

	log.Printf("AllocateIP response: %v", resp)
	return resp.Ip, resp.Gateway, nil

}

func ReleaseIPFunc(containerID string) error {
	conn, err := grpc.Dial(
		"unix://"+PPNBSocketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  50 * time.Millisecond,
				Multiplier: 1.5,
				MaxDelay:   1 * time.Second,
			},
			MinConnectTimeout: 3 * time.Second,
		}),
	)
	if err != nil {
		log.Println("did not connect: %v\n", err)
		return err
	}
	defer conn.Close()

	client := pb.NewIPAMClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	resp, err := client.ReleaseIP(ctx, &pb.ReleaseIPRequest{
		ContainerId: containerID,
	})
	if err != nil {
		log.Println("Release IP failed: %v", err)
		return err
	}

	log.Printf("AllocateIP response: %v", resp.Message)
	return nil

}

package cni

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	path2 "path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Frank-svg-dev/ppnb-cni/pkg/global"
	"github.com/Frank-svg-dev/ppnb-cni/pkg/ipam"
	pb "github.com/Frank-svg-dev/ppnb-cni/rpc"
	"github.com/Frank-svg-dev/ppnb-cni/utils"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"google.golang.org/grpc"
)

const PPNBSocketPath = "/var/run/ppnb.sock"
const PPNBIPAM_FILE_PATH string = "/var/lib/ppnb/cni/"
const PPNBCNIVethDefaultGateway = "169.254.222.0/32"
const PPNBIPAM_CACHE_PATH = "/var/run/ppnb/"

type server struct {
	pb.UnimplementedIPAMServer
	//ipPool map[string]string
}

func (s *server) AllocateIP(ctx context.Context, req *pb.AllocateIPRequest) (*pb.AllocateIPResponse, error) {
	podIP, err := utils.RandomPickAndRemove(PPNBIPAM_FILE_PATH)
	if err != nil {
		return nil, err
	}

	portInfo, err := ports.Get(ctx, global.AppConfig.NetworkClient, global.AppConfig.DataEthPort).Extract()

	if err != nil {
		return nil, err
	}

	for _, aap := range portInfo.AllowedAddressPairs {
		if aap.IPAddress == podIP {

			path := ipam.IPAM_CACHE_PATH + podIP
			_, err = os.Create(path)
			if err != nil {
				return nil, err
			}

			err = utils.WriteAndSyncFile(path, []byte(req.ContainerId), 0777)
			if err != nil {
				fmt.Println("为IP地址写入容器ID失败: err:", err.Error())
				return nil, err
			}

			return &pb.AllocateIPResponse{
				Ip:      podIP + "/32",
				Gateway: PPNBCNIVethDefaultGateway,
			}, nil
		}
	}

	rand.Seed(time.Now().UnixNano()) // 初始化随机种子
	portName := fmt.Sprintf("%05d", rand.Intn(100000))
	_, err = ports.Create(ctx, global.AppConfig.NetworkClient, ports.CreateOpts{
		NetworkID:   global.AppConfig.NetworkID,
		Name:        "ppnbcni-podip-" + portName,
		DeviceID:    global.AppConfig.DataEthPort,
		DeviceOwner: "network:secondary",
		FixedIPs: []ports.IP{
			{
				SubnetID:  global.AppConfig.SubnetID,
				IPAddress: podIP,
			},
		},
	}).Extract()

	path := PPNBIPAM_CACHE_PATH + podIP
	_, err = os.Create(path)
	if err != nil {
		return nil, err
	}

	err = utils.WriteAndSyncFile(path, []byte(req.ContainerId), 0777)
	if err != nil {
		fmt.Println("为IP地址写入容器ID失败: err:", err.Error())
		return nil, err
	}

	return &pb.AllocateIPResponse{
		Ip:      podIP + "/32",
		Gateway: PPNBCNIVethDefaultGateway,
	}, nil
}

func (s *server) ReleaseIP(ctx context.Context, req *pb.ReleaseIPRequest) (*pb.ReleaseIPResponse, error) {
	err := filepath.Walk(PPNBIPAM_CACHE_PATH, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		data, _ := os.ReadFile(path)
		if strings.Contains(string(data), req.ContainerId) {
			ipAddress := filepath.Base(path)
			podIPPath := path2.Base(path)
			podIP := podIPPath + "/32"
			fmt.Println("podIP:", podIP)

			err = DelFromIpRule(ipAddress + "/32")
			if err != nil {
				fmt.Println("删除from ip rule失败, err: ", err.Error())
				return err
			}

			err = DelToIpRule(ipAddress + "/32")
			if err != nil {
				fmt.Println("删除 to ip rule 失败, err: ", err.Error())
				return err
			}

			fmt.Println(path)
			err = os.Remove(path)
			if err != nil {
				fmt.Println("清理ip 缓存文件失败, err:   , podpath: ", err.Error(), path)
				return err
			}

			fmt.Println("podIPpath: ", podIPPath)
			fmt.Println("ipaddress: ", ipAddress)
			_, err := os.Create(PPNBIPAM_FILE_PATH + ipAddress)
			if err != nil {
				fmt.Println("restore not use ip failed, err: ", err.Error())
				return err
			}
		}

		return nil
	})

	if err != nil {
		fmt.Printf("filepath.Walk() returned %v\n", err)
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

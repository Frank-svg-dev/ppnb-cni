package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/Frank-svg-dev/ppnb-cni/pkg/cni"
	"github.com/Frank-svg-dev/ppnb-cni/pkg/ipam"
	"github.com/Frank-svg-dev/ppnb-cni/utils"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	cniVersion "github.com/containernetworking/cni/pkg/version"
	"github.com/containernetworking/plugins/pkg/ns"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
)

type NetConf struct {
	CNIVersion string `json:"cniVersion"`
	SubnetId   string `json:"subnet"`
	NetworkId  string `json:"network"`
	DeviceId   string `json:"device"`
	DataEthMac string `json:"dataEthMac"`
}

func loadNetConf(bytes []byte) (*NetConf, error) {
	n := &NetConf{}
	if err := json.Unmarshal(bytes, n); err != nil {
		return nil, err
	}

	return n, nil

}

func cmdAdd(args *skel.CmdArgs) error {

	conf, err := loadNetConf(args.StdinData)
	if err != nil {
		return err
	}
	//
	//networkClient, err := utils.GetOpenStackNetworkClient()
	//if err != nil {
	//	fmt.Println("获取Openstack客户端失败, err: ", err.Error())
	//	return err
	//}

	podIP, err := cni.IpApplicationFunc(args.ContainerID)

	if err != nil {
		fmt.Println("申请IP失败, err: ", err.Error())
		return err
	}

	//podIP, err := ipam.GetNodePodIp(networkClient, conf.DeviceId, conf.NetworkId, conf.DataEthMac, conf.SubnetId, args)
	//if err != nil {
	//	fmt.Println("申请IP失败, err: ", err.Error())
	//	return err
	//}

	ifName := args.IfName

	netns, err := ns.GetNS(args.Netns)

	if err != nil {
		return err
	}

	err = utils.CreateBridgeAndCreateVethAndSetNetworkDeviceStatusAndSetVethMaster("169.254.222.0/32", ifName, podIP, 1500, netns)
	if err != nil {
		fmt.Println("执行创建网桥, 创建 veth 设备, 添加默认路由等操作失败, err: ", err.Error())
		return err
	}

	_gw := net.ParseIP("169.254.222.0/32")

	_, _podIP, _ := net.ParseCIDR(podIP)

	result := &current.Result{
		CNIVersion: conf.CNIVersion,
		IPs: []*current.IPConfig{
			{
				Address: *_podIP,
				Gateway: _gw,
			},
		},
	}

	// 把这个结构体打印到标准输出中
	err = types.PrintResult(result, conf.CNIVersion)
	if err != nil {
		fmt.Println("结构体打出失败, err: ", err.Error())
		return err
	}

	return nil
}

func cmdCheck(args *skel.CmdArgs) error {
	return nil
}

func cmdDel(args *skel.CmdArgs) error {

	err := cni.ReleaseIPFunc(args.ContainerID)
	if err != nil {
		fmt.Println("释放IP失败, err: ", err.Error())
		return err
	}

	return nil
}

func main() {

	conn, err := grpc.Dial(
		"unix://"+ipam.PPNBSocketPath,
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
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, cniVersion.PluginSupports("0.1.0", "0.2.0", "0.3.0", "0.3.1", "0.4.0", "1.0.0"), "PPNB CNI plugin Version : 0.3.1")
}

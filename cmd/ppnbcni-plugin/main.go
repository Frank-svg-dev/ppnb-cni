package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	path2 "path"
	"path/filepath"
	"strings"

	"github.com/Frank-svg-dev/ppnb-cni/utils"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	cniVersion "github.com/containernetworking/cni/pkg/version"
	"github.com/containernetworking/plugins/pkg/ns"
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

	networkClient, err := utils.GetOpenStackNetworkClient()
	if err != nil {
		fmt.Println("获取Openstack客户端失败, err: ", err.Error())
		return err
	}

	podIP, err := utils.GetNodePodIp(networkClient, conf.DeviceId, conf.NetworkId, conf.DataEthMac, conf.SubnetId, args)
	if err != nil {
		fmt.Println("申请IP失败, err: ", err.Error())
		return err
	}

	ifName := args.IfName

	netns, err := ns.GetNS(args.Netns)

	if err != nil {
		return err
	}

	err = utils.CreateBridgeAndCreateVethAndSetNetworkDeviceStatusAndSetVethMaster("169.254.222.0/32", ifName, podIP, 1450, netns)
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

	//fmt.Fprintln(os.Stderr, "DEBUG: result = ", result)

	// 把这个结构体打印到标准输出中
	err = types.PrintResult(result, conf.CNIVersion)
	if err != nil {
		fmt.Println("结构体打出失败, err: ", err.Error())
	}

	return nil
}

func cmdCheck(args *skel.CmdArgs) error {
	return nil
}

func cmdDel(args *skel.CmdArgs) error {
	dir := "/run/ppnb/"
	keyword := args.ContainerID

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		data, _ := os.ReadFile(path)
		if strings.Contains(string(data), keyword) {
			podIP := path2.Base(path) + "/32"

			err = utils.DelFromIpRule(podIP)
			if err != nil {
				fmt.Println("删除from ip rule失败, err: ", err.Error())
				return err
			}

			err = utils.DelToIpRule(podIP)
			if err != nil {
				fmt.Println("删除 to ip rule 失败, err: ", err.Error())
				return err
			}

			err = os.Remove(path)
			if err != nil {
				fmt.Println("清理ip 缓存文件失败, err:   , podpath: ", err.Error(), path)
				return err
			}
		}

		return nil
	})

	if err != nil {
		fmt.Printf("filepath.Walk() returned %v\n", err)
		return err
	}

	return nil
}

func main() {
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, cniVersion.PluginSupports("0.1.0", "0.2.0", "0.3.0", "0.3.1", "0.4.0", "1.0.0"), "PPNB CNI plugin Version : 0.3.1")
}

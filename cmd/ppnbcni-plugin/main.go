package main

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/Frank-svg-dev/ppnb-cni/pkg/cni"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	cniVersion "github.com/containernetworking/cni/pkg/version"
	"github.com/containernetworking/plugins/pkg/ns"
)

type NetConf struct {
	CNIVersion string `json:"cniVersion"`
}

const PPNBCniDefaultMTU = 1500

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

	podIP, gwIP, err := cni.IpApplicationFunc(args.ContainerID)

	if err != nil {
		fmt.Println("申请IP失败, err: ", err.Error())
		return err
	}

	netNs, err := ns.GetNS(args.Netns)

	if err != nil {
		return err
	}

	err = cni.CreateBridgeAndCreateVethAndSetNetworkDeviceStatusAndSetVethMaster(gwIP, args.IfName, podIP, PPNBCniDefaultMTU, netNs)
	if err != nil {
		fmt.Println("执行创建网桥, 创建 veth 设备, 添加默认路由等操作失败, err: ", err.Error())
		return err
	}

	_gw := net.ParseIP(gwIP)

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
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, cniVersion.PluginSupports("0.1.0", "0.2.0", "0.3.0", "0.3.1", "0.4.0", "1.0.0"), "PPNB CNI plugin Version : 0.3.1")
}

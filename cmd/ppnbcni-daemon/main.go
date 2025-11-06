package main

import (
	"log"

	"github.com/Frank-svg-dev/ppnb-cni/pkg/ipam"
	"github.com/Frank-svg-dev/ppnb-cni/utils"
)

func main() {
	//初始化k8s客户端
	kubeClient := utils.InitNodeNetworkCRClient()
	networkClient, err := utils.GetOpenStackNetworkClient()
	if err != nil {
		log.Fatal(" 初始化k8s或openstack客户端失败  %v\n", err)
	}
	//获取虚机内部Instance_id与hostname，用于创建NodeNetwork
	instanceID, hostname := utils.GetInstanceUUID()

	//创建中继veth
	if err := utils.InitHostVethPair(); err != nil {
		log.Fatal(err)
	}

	//获取本节点NodeNetworkName CR资源
	nodeNetworkName := utils.InitNodeNetworkCR(kubeClient, instanceID, hostname)

	//初始化IPAM
	dataEthIP, err := ipam.InitNodeIPAM(kubeClient, nodeNetworkName)
	if err != nil {
		log.Fatal(err)
	}

	//给节点挂一个数据网卡
	dataEthMac, dataEthGwIP, err := utils.InitNodeNetworkPortAttachToWorker(networkClient, dataEthIP, hostname, instanceID)
	if err != nil {
		log.Fatal(err)
	}

	//获取数据网卡的eth名称
	dataEthName, err := utils.GetInterfaceByMAC(dataEthMac)
	if err != nil {
		log.Fatal(err)
	}

	//给数据网卡配置IP地址与路由表
	if err := utils.InitDataEth(dataEthIP+"/32", dataEthGwIP, dataEthName); err != nil {
		log.Fatal(err)
	}
	log.Println("初始化完成.....")

	log.Printf("gRPC IPAM server starting on unix socket ")
	if err := ipam.StartIPAMServer(); err != nil {
		log.Fatal(err)
	}
}

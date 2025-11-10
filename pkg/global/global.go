package global

import (
	"log"

	"github.com/Frank-svg-dev/pam-cni/ClientSet/clientset/versioned"
	"github.com/Frank-svg-dev/ppnb-cni/utils"
	"github.com/gophercloud/gophercloud/v2"
)

type PPNBCNIIPAMServe struct {
	NodeNetworkClient *versioned.Clientset
	ComputeClient     *gophercloud.ServiceClient
	NetworkClient     *gophercloud.ServiceClient
	InstanceID        string
	NetworkID         string
	SubnetID          string
	DataEthIP         string
	DataEthMAC        string
	DataEthPort       string
	DataEthGw         string
}

const (
	PPNBSocketPath            = "/var/run/ppnb.sock"
	PPNBIPAM_FILE_PATH        = "/var/lib/ppnb/cni/"
	PPNBCNIVethDefaultGateway = "169.254.222.0/32"
	PPNBIPAM_CACHE_PATH       = "/var/run/ppnb/"
)

var AppConfig *PPNBCNIIPAMServe

func NewPPNBCNIIPAMService(networkID, subnetID, securityGroupsID string) *PPNBCNIIPAMServe {
	//初始化k8s客户端
	kubeClient := utils.InitNodeNetworkCRClient()
	networkClient, err := utils.GetOpenStackNetworkClient()
	if err != nil {
		log.Fatal(" 初始化openstack neutron客户端失败  %v\n", err)
	}

	computeClient, err := utils.GetOpenStackComputeClient()
	if err != nil {
		log.Fatal(" 初始化openstack nova 客户端失败  %v\n", err)
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
	dataEthIP, err := InitNodeIPAM(kubeClient, nodeNetworkName)
	if err != nil {
		log.Fatal(err)
	}

	//给节点挂一个数据网卡
	dataEthMac, dataEthGwIP, dataEthPort, err := utils.InitNodeNetworkPortAttachToWorker(networkClient, dataEthIP, hostname, instanceID, networkID, subnetID, securityGroupsID)
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
	return &PPNBCNIIPAMServe{
		NodeNetworkClient: kubeClient,
		ComputeClient:     computeClient,
		NetworkClient:     networkClient,
		InstanceID:        instanceID,
		NetworkID:         networkID,
		SubnetID:          subnetID,
		DataEthIP:         dataEthIP,
		DataEthMAC:        dataEthMac,
		DataEthPort:       dataEthPort,
		DataEthGw:         dataEthGwIP,
	}
}

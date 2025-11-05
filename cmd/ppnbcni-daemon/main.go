package main

import (
	"fmt"
	"os"

	"github.com/Frank-svg-dev/ppnb-cni/utils"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/attachinterfaces"
)

func main() {
	instanceID, hostname := utils.GetInstanceUUID()

	dyClient := utils.InitKubernetesClient()

	crName := utils.CreateNodeNetworkCR(dyClient, instanceID, hostname)

	Cidr := utils.GetNodeCidrNetwork(crName, dyClient)

	err := utils.CreateIpamFile(Cidr)
	if err != nil {
		panic(err)
	}

	networkClient, err := utils.GetOpenStackNetworkClient()
	if err != nil {
		panic(err)
	}

	//创建一个port， 挑一个IP， 然后从/var/lib/ppnb/cni里删除掉

	err = os.Remove(utils.IPAM_FILE_PATH + "123123")
	if err != nil {
		panic(err)
	}

	computeClient, err := utils.GetOpenStackComputeClient()

	if err != nil {
		panic(err)
	}
	//挂载Port就此一次
	fmt.Println(networkClient, computeClient)

	//根据cidr， 以及挂载port返回的网卡名，
	attachOpts := attachinterfaces.CreateOpts{
		PortID: p.ID,
		// 或者可以用 NetworkID（部分环境支持），比如:
		// NetworkID: networkID,
		// FixedIP: &attachinterfaces.FixedIP{SubnetID: "subnet-uuid", IPAddress: "10.0.0.50"},
	}

	iface, err := attachinterfaces.Create(computeClient, serverID, attachOpts).Extract()
	//以上步骤就此一次

}

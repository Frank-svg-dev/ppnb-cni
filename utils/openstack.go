package utils

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/attachinterfaces"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
)

func GetOpenStackNetworkClient() (*gophercloud.ServiceClient, error) {
	ctx := context.Background()

	//providerClient, err := openstack.AuthenticatedClient(ctx, gophercloud.AuthOptions{
	//	ApplicationCredentialName:   os.Getenv("OS_APPLICATION_CREDENTIAL_ID"),
	//	ApplicationCredentialSecret: os.Getenv("OS_APPLICATION_CREDENTIAL_SECRET"),
	//	IdentityEndpoint:            "http://keystone.openstack.svc.cluster.local/v3",
	//	//Username:                    "admin",
	//	//Password:                    "Admin@OPS20!8",
	//	//DomainName:                  "Default",
	//	//TenantName:                  "admin",
	//})

	opts, err := openstack.AuthOptionsFromEnv()
	providerClient, err := openstack.AuthenticatedClient(ctx, opts)

	if err != nil {
		return nil, err
	}

	networkClient, err := openstack.NewNetworkV2(providerClient, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})

	if err != nil {
		return nil, err
	}

	return networkClient, nil
}

func GetOpenStackComputeClient() (*gophercloud.ServiceClient, error) {
	ctx := context.Background()
	opts, err := openstack.AuthOptionsFromEnv()
	providerClient, err := openstack.AuthenticatedClient(ctx, opts)
	if err != nil {
		return nil, err
	}

	computeClient, err := openstack.NewComputeV2(providerClient, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})

	if err != nil {
		return nil, err
	}

	return computeClient, nil
}

func InitNodeNetworkPortAttachToWorker(networkClient *gophercloud.ServiceClient, ipaddress, hostname, instanceId string, networkID, subnetID, securityGroupsID string) (string, string, string, error) {
	ctx := context.Background()

	computeClient, err := GetOpenStackComputeClient()
	if err != nil {
		log.Fatal(err)
	}

	subnetInfo, err := subnets.Get(ctx, networkClient, subnetID).Extract()
	if err != nil {
		log.Fatal(err)
		return "", "", "", err
	}

	dataPort, err := ports.Create(ctx, networkClient, ports.CreateOpts{
		NetworkID:      networkID,
		Name:           "ppnb-vmport-" + hostname,
		SecurityGroups: &[]string{securityGroupsID},
		FixedIPs: []ports.IP{
			{
				SubnetID:  subnetID,
				IPAddress: ipaddress,
			},
		},
	}).Extract()
	if err != nil {
		if ue, ok := err.(gophercloud.ErrUnexpectedResponseCode); ok && ue.Actual != 409 {
			return "", "", "", err
		}

		allPages, err := ports.List(networkClient, ports.ListOpts{
			NetworkID: networkID,
			FixedIPs: []ports.FixedIPOpts{
				{
					SubnetID:  subnetID,
					IPAddress: ipaddress,
				},
			},
		}).AllPages(ctx)

		if err != nil {
			log.Println("获取数据网卡port列表失败: err:", err.Error())
			return "", "", "", err
		}

		allPorts, err := ports.ExtractPorts(allPages)
		if err != nil {
			log.Println("获取数据网卡port信息失败: err:", err.Error())
			return "", "", "", err
		}

		if len(allPorts) == 0 {
			return "", "", "", errors.New(fmt.Sprintf("no port found with IP %s", ipaddress))
		}

		return allPorts[0].MACAddress, subnetInfo.GatewayIP, allPorts[0].ID, nil
	}

	_, err = attachinterfaces.Create(ctx, computeClient, instanceId, attachinterfaces.CreateOpts{
		PortID: dataPort.ID,
	}).Extract()

	if err != nil {
		return "", "", "", err
	}

	return dataPort.MACAddress, subnetInfo.GatewayIP, dataPort.ID, nil
}

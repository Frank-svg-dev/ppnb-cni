package utils

import (
	"context"
	"log"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/attachinterfaces"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
)

func GetOpenStackNetworkClient() (*gophercloud.ServiceClient, error) {
	ctx := context.Background()
	providerClient, err := openstack.AuthenticatedClient(ctx, gophercloud.AuthOptions{
		IdentityEndpoint: "http://keystone.openstack.svc.cluster.local/v3",
		Username:         "admin",
		Password:         "Admin@OPS20!8",
		DomainName:       "Default",
		TenantName:       "admin",
	})
	if err != nil {
		return nil, err
	}

	networkClient, err := openstack.NewNetworkV2(providerClient, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})

	if err != nil {
		return nil, err
	}

	return networkClient, nil
}

func GetOpenStackComputeClient() (*gophercloud.ServiceClient, error) {
	ctx := context.Background()
	providerClient, err := openstack.AuthenticatedClient(ctx, gophercloud.AuthOptions{
		IdentityEndpoint: "http://keystone.openstack.svc.cluster.local/v3",
		Username:         "admin",
		Password:         "Admin@OPS20!8",
		DomainName:       "Default",
		TenantName:       "admin",
	})
	if err != nil {
		return nil, err
	}

	computeClient, err := openstack.NewComputeV2(providerClient, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})

	if err != nil {
		return nil, err
	}

	return computeClient, nil
}

func InitNodeNetworkPortAttachToWorker(networkClient *gophercloud.ServiceClient, ipaddress, hostname, instanceId string) (string, string, error) {
	ctx := context.Background()

	computeClient, err := GetOpenStackComputeClient()
	if err != nil {
		log.Fatal(err)
	}

	dataPort, err := ports.Create(ctx, networkClient, ports.CreateOpts{
		NetworkID:      "xxxxxx",
		Name:           "ppnb-vmport-" + hostname,
		SecurityGroups: &[]string{"125411c2-a6f1-4ca7-a8cc-f5c050a32485"},
		FixedIPs: []ports.IP{
			{
				SubnetID:  "xxxxxxxxxxx",
				IPAddress: ipaddress,
			},
		},
	}).Extract()
	if err != nil {
		return "", "", err
	}

	subnetInfo, err := subnets.Get(ctx, networkClient, "xxxxxxxxxxx").Extract()
	if err != nil {
		log.Fatal(err)
		return "", "", err
	}

	_, err = attachinterfaces.Create(ctx, computeClient, instanceId, attachinterfaces.CreateOpts{
		PortID: dataPort.ID,
	}).Extract()

	if err != nil {
		return "", "", err
	}

	return dataPort.MACAddress, subnetInfo.GatewayIP, nil

}

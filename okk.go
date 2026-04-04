package main

import (
	"context"
	"fmt"
	"net"

	"github.com/Frank-svg-dev/ppnb-cni/utils"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
)

func main() {
	client, err := utils.GetOpenStackNetworkClient()
	if err != nil {
		fmt.Println(err)
	}
	allPages, err := subnets.List(client, subnets.ListOpts{}).AllPages(context.Background())
	if err != nil {
		fmt.Println(err)
	}
	allSubnets, err := subnets.ExtractSubnets(allPages)
	if err != nil {
		fmt.Println(err)
	}
	for _, subnet := range allSubnets {
		fmt.Println(subnet)
	}
}

// inc() 用来自增 IP 地址
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

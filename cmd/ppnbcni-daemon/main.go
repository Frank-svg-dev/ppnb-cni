package main

import (
	"flag"
	"log"

	"github.com/Frank-svg-dev/ppnb-cni/pkg/global"
	"github.com/Frank-svg-dev/ppnb-cni/pkg/ipam"
)

func main() {
	var networkID string
	flag.StringVar(&networkID, "network-id", "", "pod network id")

	var subnetID string
	flag.StringVar(&subnetID, "subnet-id", "", "pod subnet id")

	var securityGroupsID string
	flag.StringVar(&securityGroupsID, "security-group-id", "", "pod数据网卡安全组ID")

	flag.Parse()

	global.AppConfig = global.NewPPNBCNIIPAMService(networkID, subnetID, securityGroupsID)

	log.Printf("gRPC IPAM server starting on unix socket ")
	if err := ipam.StartIPAMServer(); err != nil {
		log.Fatal(err)
	}
}

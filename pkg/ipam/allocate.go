package ipam

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/Frank-svg-dev/ppnb-cni/utils"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
)

func newPodIP(networkClient *gophercloud.ServiceClient,
	netId, deviceId, dataMac, subentId string, allowPair []ports.AddressPair) (string, error) {
	ctx := context.Background()

	//subnet, err := subnets.Get(ctx, networkClient, id).Extract()
	//if err != nil {
	//	return subnets.Subnet{}
	//}
	rand.Seed(time.Now().UnixNano()) // 初始化随机种子
	portname := fmt.Sprintf("%05d", rand.Intn(100000))

	port, err := ports.Create(ctx, networkClient, ports.CreateOpts{
		NetworkID:   netId,
		Name:        "ppnb-podip-" + portname,
		DeviceID:    deviceId,
		DeviceOwner: "network:secondary",
		//SecurityGroups: &[]string{"125411c2-a6f1-4ca7-a8cc-f5c050a32485"},
		FixedIPs: []ports.IP{
			{
				SubnetID: subentId,
			},
		},
	}).Extract()
	if err != nil {
		return "", err
	}

	ap := ports.AddressPair{
		IPAddress:  port.FixedIPs[0].IPAddress,
		MACAddress: dataMac,
	}

	allowPair = append(allowPair, ap)

	err = ports.Update(ctx, networkClient, deviceId, ports.UpdateOpts{AllowedAddressPairs: &allowPair}).Err
	if err != nil {
		return "", err
	}

	return port.FixedIPs[0].IPAddress, nil
}

func GetNodePodIp(networkClient *gophercloud.ServiceClient, portId, netId, dataMac, subentId string, containerID string) (string, error) {
	ctx := context.Background()

	portsList, err := ports.Get(ctx, networkClient, portId).Extract()
	if err != nil {
		return "", err
	}

	entries, err := os.ReadDir(IPAM_CACHE_PATH)
	if err != nil {
		return "", err
	}

	var localIPs []string

	for _, entry := range entries {
		if !entry.IsDir() {
			localIPs = append(localIPs, entry.Name())
		}
	}

	localIp := makeSet(localIPs)

	for i := 0; i < len(portsList.AllowedAddressPairs); i++ {
		if _, ok := localIp[portsList.AllowedAddressPairs[i].IPAddress]; ok {
			continue
		} else {
			podIP := portsList.AllowedAddressPairs[i].IPAddress

			path := IPAM_CACHE_PATH + podIP
			_, err = os.Create(path)
			if err != nil {
				return "", err
			}

			err = utils.WriteAndSyncFile(path, []byte(containerID), 0777)
			if err != nil {
				fmt.Println("为IP地址写入容器ID失败: err:", err.Error())
				return "", err
			}

			return podIP + "/32", nil
		}
	}
	//netId, deviceId, dataMac, subentId string,
	PodIP, err := newPodIP(networkClient, netId, portId, dataMac, subentId, portsList.AllowedAddressPairs)
	if err != nil {
		return "", err
	}

	path := IPAM_CACHE_PATH + PodIP
	_, err = os.Create(path)
	if err != nil {
		return "", err
	}

	err = utils.WriteAndSyncFile(path, []byte(containerID), 0777)
	if err != nil {
		fmt.Println("为IP地址写入容器ID失败: err:", err.Error())
		return "", err
	}

	return PodIP + "/32", nil
}

func makeSet(list []string) map[string]struct{} {
	set := make(map[string]struct{}, len(list))
	for _, v := range list {
		set[v] = struct{}{}
	}
	return set
}

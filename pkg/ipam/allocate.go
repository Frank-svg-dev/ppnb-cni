package ipam

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	path2 "path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Frank-svg-dev/ppnb-cni/utils"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
)

func allocateIP(containerID string) (string, error) {
	podIP, err := utils.RandomPickAndRemove(IPAM_FILE_PATH)
	if err != nil {
		return "", err
	}

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
	return podIP, nil

}
func releaseIP(containerID string) error {
	err := filepath.Walk(IPAM_CACHE_PATH, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		data, _ := os.ReadFile(path)
		if strings.Contains(string(data), containerID) {
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

			_, err := os.Create(IPAM_FILE_PATH + podIP)
			if err != nil {
				fmt.Println("restore not use ip failed, err: ", err.Error())
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

func GetNodePodIp(networkClient *gophercloud.ServiceClient, portId, netId, dataMac, subentId string, args *skel.CmdArgs) (string, error) {
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

			err = utils.WriteAndSyncFile(path, []byte(args.ContainerID), 0777)
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

	err = utils.WriteAndSyncFile(path, []byte(args.ContainerID), 0777)
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

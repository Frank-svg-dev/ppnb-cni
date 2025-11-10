package ipam

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	nodeNetworkClientSet "github.com/Frank-svg-dev/pam-cni/ClientSet/clientset/versioned"
	"github.com/Frank-svg-dev/ppnb-cni/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	IPAM_FILE_PATH  = "/var/lib/ppnb/cni/"
	IPAM_CACHE_PATH = "/var/run/ppnb/"
)

func InitNodeIPAM(nnClient *nodeNetworkClientSet.Clientset, nodeNetworkName string) (string, error) {

	for i := 0; i < 30; i++ {
		nodeNetwork, err := nnClient.NetworkV1alpha1().NodeNetworks().Get(context.Background(), nodeNetworkName, metav1.GetOptions{})

		if err != nil {
			if !apierrors.IsNotFound(err) {
				log.Println("failed to get NodeNetwork CR: %v", err)
				return "", err
			}
		}

		if nodeNetwork.Spec.CIDR != "" {
			err := CreateIpamFile(nodeNetwork.Spec.CIDR)
			if err != nil {
				log.Println("failed to create NodeNetwork local ip pool : %v", err)
				return "", err
			}

			dataEthIpaddress := utils.FirstIPFromCIDRStr(nodeNetwork.Spec.CIDR)
			if dataEthIpaddress == "" {
				return "", errors.New("failed to find IP address from CIDR")
			}

			err = os.Remove(IPAM_FILE_PATH + dataEthIpaddress)
			if err != nil {
				log.Println("failed to remove IP address from file : %v", err)
				return "", err
			}

			return dataEthIpaddress, nil
		}

		time.Sleep(1 * time.Second)
	}

	return "", errors.New("failed to find NodeNetwork CR")
}

func CreateIpamFile(cidr string) error {
	err := os.MkdirAll(IPAM_FILE_PATH, 0777)
	if err != nil {
		if !os.IsExist(err) {
			return err
		}
	}

	// 1️⃣ 解析 CIDR
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(err)
	}

	// 用 ipnet 的网络部分
	ip = ip.Mask(ipnet.Mask)

	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}

	// 去掉网络地址和广播地址
	if len(ips) > 2 {
		ips = ips[1 : len(ips)-1]
	}

	for _, addr := range ips {
		_, err := os.Create(filepath.Join(IPAM_FILE_PATH, addr))
		if err != nil {
			if os.IsExist(err) {
				continue
			}
			return err
		}

	}

	return nil
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

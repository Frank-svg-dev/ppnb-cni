package utils

import (
	"net"
	"os"
	"path/filepath"
)

const (
	IPAM_FILE_PATH = "/var/lib/ppnb/cni/"
)

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

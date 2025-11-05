package main

import (
	"fmt"
	"net"
)

func main() {
	cidr := "10.10.1.1/24"

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
		fmt.Println(addr)
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

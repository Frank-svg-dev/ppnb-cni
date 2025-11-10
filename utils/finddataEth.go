package utils

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/vishvananda/netlink"
)

func GetInterfaceByMAC(targetMAC string) (string, error) {
	// 标准化 MAC 地址格式
	targetMAC = strings.ToLower(strings.ReplaceAll(targetMAC, ":", ""))
	targetMAC = strings.ReplaceAll(targetMAC, "-", "")

	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to get network interfaces: %v", err)
	}

	for _, iface := range interfaces {
		// 获取网卡的 MAC 地址
		macAddr := iface.HardwareAddr.String()
		macAddr = strings.ToLower(strings.ReplaceAll(macAddr, ":", ""))
		macAddr = strings.ReplaceAll(macAddr, "-", "")

		if macAddr == targetMAC {
			return iface.Name, nil
		}
	}

	return "", fmt.Errorf("interface with MAC %s not found", targetMAC)
}

func InitHostVethPair() error {
	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name: "ppnb_host", // 主接口名
			MTU:  1500,        // 可选
		},
		PeerName: "ppnb_net", // 对端接口名
	}

	// 创建 veth pair
	if err := netlink.LinkAdd(veth); err != nil {
		if !os.IsExist(err) {
			log.Println("创建 veth 失败: %v", err)
			return err
		}
	}

	// 启动接口（等价于 ip link set up）
	hostLink, _ := netlink.LinkByName("ppnb_host")
	peerLink, _ := netlink.LinkByName("ppnb_net")

	addr, err := netlink.ParseAddr("169.254.222.0/32")
	if err != nil {
		log.Println("解析 IP 失败: %v", err)
		return err
	}
	if err := netlink.AddrAdd(hostLink, addr); err != nil {

		if !os.IsExist(err) {

			log.Println("配置 IP 失败: %v", err)
			return err
		}

	}

	if err := netlink.LinkSetUp(hostLink); err != nil {
		log.Println("启动 %s 失败: %v", "ppnb_host", err)
		return err
	}
	if err := netlink.LinkSetUp(peerLink); err != nil {

		log.Println("启动 %s 失败: %v", "ppnb_net", err)
		return err
	}
	fmt.Println("✅ 成功创建 veth pair: ppnb_host <-> ppnb_net, 并配置IP地址")
	return nil
}

func InitDataEth(ipCIDR, gw, linkName string) error {
	link, err := netlink.LinkByName(linkName)
	if err != nil {
		log.Println(fmt.Errorf("找不到网卡 %s: %w", linkName, err))
		return err
	}

	// 2. 解析 IP 地址
	addr, err := netlink.ParseAddr(ipCIDR)
	if err != nil {
		log.Println(fmt.Errorf("解析 CIDR 失败: %w", err))
		return err
	}

	// 3. 添加 IP 地址
	if err := netlink.AddrAdd(link, addr); err != nil {
		if !os.IsExist(err) {
			log.Println(fmt.Errorf("添加 IP 地址失败: %w", err))
			return err
		}
	}

	_, localNet, err := net.ParseCIDR(gw + "/32")
	if err != nil {
		log.Println(fmt.Errorf("解析目标网段失败: %w", err))
		return err
	}

	localRoute := &netlink.Route{
		LinkIndex: link.Attrs().Index,
		Scope:     netlink.SCOPE_LINK,
		Dst:       localNet,
		Table:     50,
	}

	if err := netlink.RouteAdd(localRoute); err != nil {
		if !os.IsExist(err) {
			log.Println(fmt.Errorf("添加本地路由失败: %w", err))
			return err
		}

	}

	// 2. 构造路由目标
	_, dstNet, err := net.ParseCIDR("0.0.0.0/0")
	if err != nil {
		log.Println(fmt.Errorf("解析目标网段失败: %w", err))
		return err
	}

	gwIP := net.ParseIP(gw)
	fmt.Println(gw)

	// 3. 构造并添加路由
	route := &netlink.Route{
		LinkIndex: link.Attrs().Index,
		Dst:       dstNet,
		Gw:        gwIP,
		Scope:     netlink.SCOPE_UNIVERSE,
		Table:     50,
	}

	if err := netlink.RouteAdd(route); err != nil {
		if !os.IsExist(err) {
			log.Println(fmt.Errorf("添加default路由失败: %w", err))
			return err
		}
	}
	return nil
}

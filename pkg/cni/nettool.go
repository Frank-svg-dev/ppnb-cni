package cni

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/Frank-svg-dev/ppnb-cni/pkg/global"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

func delInterfaceByName(name string) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return err
	}

	if err = netlink.LinkDel(link); err != nil {
		return fmt.Errorf("failed to delete interface: %v", err)
	}

	return nil
}

func SetUpVeth(veth ...*netlink.Veth) error {
	for _, v := range veth {
		// 启动 veth 设备
		err := netlink.LinkSetUp(v)
		if err != nil {
			fmt.Println("启动 veth1 失败, err: ", err.Error())
			return err
		}
	}
	return nil
}

func CreateVethPair(ifName string, mtu int, hostName ...string) (*netlink.Veth, *netlink.Veth, error) {
	vethPairName := ""
	if len(hostName) > 0 && hostName[0] != "" {
		vethPairName = hostName[0]
	} else {
		for {
			_vname, err := RandomVethName()
			vethPairName = _vname
			if err != nil {
				fmt.Println("生成随机 veth pair 名字失败, err: ", err.Error())
				return nil, nil, err
			}

			_, err = netlink.LinkByName(vethPairName)
			if err != nil && !os.IsExist(err) {
				// 上面生成随机名字可能会重名, 所以这里先尝试按照这个名字获取一下
				// 如果没有这个名字的设备, 那就可以 break 了
				break
			}
		}
	}

	if vethPairName == "" {
		return nil, nil, errors.New("create veth pair's name error")
	}

	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name: ifName,
			// Flags:     net.FlagUp,
			MTU: mtu,
			// Namespace: netlink.NsFd(int(ns.Fd())), // 先不设置 ns, 要不一会儿下头 LinkByName 时候找不到
		},
		PeerName: vethPairName,
		// PeerNamespace: netlink.NsFd(int(ns.Fd())),
	}

	// 创建 veth pair
	err := netlink.LinkAdd(veth)

	if err != nil {
		fmt.Println("创建 veth 设备失败, err: ", err.Error())
		return nil, nil, err
	}

	// 尝试重新获取 veth 设备看是否能成功
	veth1, err := netlink.LinkByName(ifName) // veth1 一会儿要在 pod(net ns) 里
	if err != nil {
		// 如果获取失败就尝试删掉
		netlink.LinkDel(veth1)
		fmt.Println("创建完 veth 但是获取失败, err: ", err.Error())
		return nil, nil, err
	}

	// 尝试重新获取 veth 设备看是否能成功
	veth2, err := netlink.LinkByName(vethPairName) // veth2 在主机上
	if err != nil {
		// 如果获取失败就尝试删掉
		netlink.LinkDel(veth2)
		fmt.Println("创建完 veth 但是获取失败, err: ", err.Error())
		return nil, nil, err
	}

	return veth1.(*netlink.Veth), veth2.(*netlink.Veth), nil
}

func DelVethPair(ifName string) error {
	return delInterfaceByName(ifName)
}

func setIpForDevice(name string, ip string, mode ...string) error {
	deviceType := ""
	if len(mode) != 0 {
		deviceType = mode[0]
	}
	link, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("failed to get %s device by name %q, error: %v", deviceType, name, err)
	}

	ipaddr, ipnet, err := net.ParseCIDR(ip)
	if err != nil {
		return fmt.Errorf("failed to transform the ip %q, error : %v", ip, err)
	}
	ipnet.IP = ipaddr
	err = netlink.AddrAdd(link, &netlink.Addr{IPNet: ipnet})
	if err != nil {
		return fmt.Errorf("can not add the ip %q to %s device %q, error: %v", ip, deviceType, name, err)
	}
	return nil
}

func SetIpForVeth(name string, podIP string) error {
	return setIpForDevice(name, podIP, "veth")
}

func SetDeviceToNS(device netlink.Link, ns ns.NetNS) error {
	err := netlink.LinkSetNsFd(device, int(ns.Fd()))
	if err != nil {
		return fmt.Errorf("failed to add the device %q to ns: %v", device.Attrs().Name, err)
	}
	return nil
}

func SetVethNsFd(veth *netlink.Veth, ns ns.NetNS) error {
	return SetDeviceToNS(veth, ns)
}

func SetDefaultRouteToVeth(gwIP net.IP, veth netlink.Link) error {
	_, gwNet, err := net.ParseCIDR(global.PPNBCNIVethDefaultGateway)
	if err != nil {
		return err
	}
	err = AddHostRoute(veth, gwNet)
	if err != nil {
		return err
	}
	err = AddDefaultRoute(gwIP, veth)
	if err != nil {
		return err
	}
	return nil
}

func SetDefaultRouteToHostVeth(podIP string, veth netlink.Link) error {
	_, podNet, _ := net.ParseCIDR(podIP)
	return AddHostRoute(veth, podNet)
}

func AddHostRoute(dev netlink.Link, ipn *net.IPNet) error {
	return netlink.RouteAdd(&netlink.Route{
		LinkIndex: dev.Attrs().Index,
		Scope:     netlink.SCOPE_LINK,
		Dst:       ipn,
	})
}

// forked from plugins/pkg/ip/route_linux.go
func AddRoute(ipn *net.IPNet, gw net.IP, dev netlink.Link, scope ...netlink.Scope) error {
	defaultScope := netlink.SCOPE_UNIVERSE
	if len(scope) > 0 {
		defaultScope = scope[0]
	}
	return netlink.RouteAdd(&netlink.Route{
		LinkIndex: dev.Attrs().Index,
		Scope:     defaultScope,
		Dst:       ipn,
		Gw:        gw,
	})
}

// forked from plugins/pkg/ip/route_linux.go
func AddDefaultRoute(gw net.IP, dev netlink.Link) error {
	_, defNet, _ := net.ParseCIDR("0.0.0.0/0")

	err := AddRoute(defNet, gw, dev)

	if err != nil {
		return err
	}

	return nil
}

// forked from /plugins/pkg/ip/link_linux.go
// RandomVethName returns string "veth" with random prefix (hashed from entropy)
func RandomVethName() (string, error) {
	entropy := make([]byte, 4)
	_, err := rand.Read(entropy)
	if err != nil {
		return "", fmt.Errorf("failed to generate random veth name: %v", err)
	}

	// NetworkManager (recent versions) will ignore veth devices that start with "veth"
	return fmt.Sprintf("ppth%x", entropy), nil
}

func DelFromIpRule(podIp string) error {
	_, src, _ := net.ParseCIDR(podIp)
	rule := netlink.NewRule()
	rule.Src = src
	rule.Table = global.PPNBIpRuleDefaultTable
	rule.Priority = global.PPNBIpRuleDefaultTablePriority
	if err := netlink.RuleDel(rule); err != nil {
		fmt.Println("删除 form ip  rule 失败:", err)
		return err
	}
	return nil
}

func DelToIpRule(podIp string) error {
	_, dst, _ := net.ParseCIDR(podIp)

	// 删除 rule
	rule := netlink.NewRule()
	rule.Dst = dst
	rule.Table = unix.RT_TABLE_MAIN
	rule.Priority = 21

	// 添加 rule
	if err := netlink.RuleDel(rule); err != nil {
		fmt.Println("删除 to ip rule 失败:", err)
		return err
	}

	return nil
}

func addFromIpRule(podIp string) error {
	//src := &net.IPNet{
	//	IP:   net.ParseIP("192.168.1.10"),
	//	Mask: net.CIDRMask(32, 32),
	//}

	_, src, _ := net.ParseCIDR(podIp)

	// 创建 rule
	rule := netlink.NewRule()
	rule.Src = src
	rule.Table = global.PPNBIpRuleDefaultTable
	rule.Priority = global.PPNBIpRuleDefaultTablePriority

	// 添加 rule
	if err := netlink.RuleAdd(rule); err != nil {
		fmt.Println("添加 form ip  rule 失败:", err)
		return err
	}

	return nil
}

func addToIpRule(podIp string) error {
	_, dst, _ := net.ParseCIDR(podIp)

	// 创建 rule
	rule := netlink.NewRule()
	rule.Dst = dst
	rule.Table = unix.RT_TABLE_MAIN
	rule.Priority = 21

	// 添加 rule
	if err := netlink.RuleAdd(rule); err != nil {
		fmt.Println("添加 to ip rule 失败:", err)
		return err
	}

	return nil
}

func CreateBridgeAndCreateVethAndSetNetworkDeviceStatusAndSetVethMaster(
	gw, ifName, podIP string, mtu int, netns ns.NetNS,
) error {

	err := netns.Do(func(hostNs ns.NetNS) error {
		// 创建一对儿 veth 设备
		containerVeth, hostVeth, err := CreateVethPair(ifName, mtu)
		if err != nil {
			fmt.Println("创建 veth 失败, err: ", err.Error())
			return err
		}

		// 把随机起名的 veth 那头放在主机上
		err = SetVethNsFd(hostVeth, hostNs)
		if err != nil {
			fmt.Println("把 veth 设置到 ns 下失败: ", err.Error())
			return err
		}

		// 然后把要被放到 pod 中的那头 veth 塞上 podIP
		err = SetIpForVeth(containerVeth.Name, podIP)
		if err != nil {
			fmt.Println("给 veth 设置 ip 失败, err: ", err.Error())
			return err
		}

		// 然后启动它
		err = SetUpVeth(containerVeth)
		if err != nil {
			fmt.Println("启动 veth pair 失败, err: ", err.Error())
			return err
		}

		//启动之后给这个 netns 设置默认路由 以便让其他网段的包也能从 veth 走到网桥
		gwNetIP, _, err := net.ParseCIDR(gw)
		if err != nil {
			fmt.Println("转换 gwip 失败, err:", err.Error())
			return err
		}

		// 给 pod(net ns) 中加一个默认路由规则, 该规则让匹配了 0.0.0.0 的都走上边创建的那个 container veth
		err = SetDefaultRouteToVeth(gwNetIP, containerVeth)
		if err != nil {
			fmt.Println("SetDefaultRouteToVeth 时出错, err: ", err.Error())
			return err
		}

		hostNs.Do(func(_ ns.NetNS) error {
			// 重新获取一次 host 上的 veth, 因为 hostVeth 发生了改变
			_hostVeth, err := netlink.LinkByName(hostVeth.Attrs().Name)
			hostVeth = _hostVeth.(*netlink.Veth)
			if err != nil {
				fmt.Println("重新获取 hostVeth 失败, err: ", err.Error())
				return err
			}
			// 启动它
			err = SetUpVeth(hostVeth)
			if err != nil {
				fmt.Println("启动 veth pair 失败, err: ", err.Error())
				return err
			}

			err = SetDefaultRouteToHostVeth(podIP, hostVeth)

			if err != nil {
				fmt.Println("SetDefaultRouteToHostVeth 时出错, err: ", err.Error())
				return err
			}

			err = addFromIpRule(podIP)
			if err != nil {
				fmt.Println("Add From Ip rule失败, err :", err.Error())
				return err
			}

			err = addToIpRule(podIP)
			if err != nil {
				fmt.Println("Add to Ip rule失败, err :", err.Error())
				return err
			}

			return nil
		})

		return nil
	})

	if err != nil {
		return err
	}
	return nil
}

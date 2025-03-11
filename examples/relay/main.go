package main

//
//import (
//	"context"
//	"fmt"
//	"log"
//	"time"
//
//	"github.com/libp2p/go-libp2p"
//	"github.com/libp2p/go-libp2p/core/host"
//	"github.com/libp2p/go-libp2p/core/network"
//	"github.com/libp2p/go-libp2p/core/peer"
//	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
//	"github.com/libp2p/go-libp2p/p2p/protocol/holepunch"
//	"github.com/libp2p/go-libp2p/p2p/protocol/identify"
//	ma "github.com/multiformats/go-multiaddr"
//)
//
//func main() {
//	// 启动程序
//	run()
//}
//
//func run() {
//	// 创建一个中继节点，作为服务端
//	fmt.Println("开始创建服务端")
//	relay1, err := libp2p.New(
//		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
//		libp2p.EnableRelayService(),
//		libp2p.EnableHolePunching(),
//	)
//	fmt.Println("服务端创建完成")
//	if err != nil {
//		log.Printf("创建服务端失败: %v", err)
//		return
//	}
//	defer relay1.Close()
//	fmt.Printf("服务端的ID是: %s，使用的端口: %d\n", relay1.ID(), getPortFromAddrs(relay1.Addrs()))
//
//	// 配置服务端（中继节点）提供中继服务
//	fmt.Println("开始配置服务端中继服务")
//	_, err = relay.New(relay1)
//	fmt.Println("服务端中继服务配置完成")
//	if err != nil {
//		log.Printf("中继服务配置失败: %v", err)
//		return
//	}
//
//	// 为服务端启用打洞服务
//	ids, err := identify.NewIDService(relay1)
//	if err != nil {
//		log.Printf("服务端identify服务配置失败: %v", err)
//		return
//	}
//	_, err = holepunch.NewService(relay1, ids, func() []ma.Multiaddr { return relay1.Addrs() })
//	if err != nil {
//		log.Printf("服务端打洞服务配置失败: %v", err)
//		return
//	}
//
//	relay1info := peer.AddrInfo{ID: relay1.ID(), Addrs: relay1.Addrs()}
//
//	// 创建两个模拟无法直接连接的主机
//	fmt.Println("开始创建客户端1")
//	unreachable1, err := createClient("客户端1", relay1info)
//	if err != nil {
//		log.Printf("创建客户端1失败: %v", err)
//		return
//	}
//	defer unreachable1.Close()
//
//	fmt.Println("开始创建客户端2")
//	unreachable2, err := createClient("客户端2", relay1info)
//	if err != nil {
//		log.Printf("创建客户端2失败: %v", err)
//		return
//	}
//	defer unreachable2.Close()
//
//	// 客户端1 连接服务端
//	fmt.Println("客户端1尝试连接服务端...")
//	if err := unreachable1.Connect(context.Background(), relay1info); err != nil {
//		log.Printf("客户端1连接服务端失败: %v", err)
//		return
//	}
//	fmt.Println("客户端1成功连接服务端")
//
//	// 延迟1秒再让客户端2进行连接
//	time.Sleep(time.Second)
//
//	// 客户端2 连接服务端
//	fmt.Println("客户端2尝试连接服务端...")
//	if err := unreachable2.Connect(context.Background(), relay1info); err != nil {
//		log.Printf("客户端2连接服务端失败: %v", err)
//		return
//	}
//	fmt.Println("客户端2成功连接服务端")
//
//	// 让客户端1和客户端2相互连接
//	fmt.Println("客户端1尝试连接客户端2...")
//	if err := unreachable1.Connect(context.Background(), peer.AddrInfo{
//		ID:    unreachable2.ID(),
//		Addrs: unreachable2.Addrs(),
//	}); err != nil {
//		log.Printf("客户端1连接客户端2失败: %v", err)
//		return
//	}
//	fmt.Println("客户端1成功连接客户端2")
//
//	// 等待一段时间，让打洞完成
//	time.Sleep(3 * time.Second)
//	fmt.Println("两个客户端已经建立连接，开始检查连接类型...")
//
//	// 检查连接类型
//	conns := unreachable1.Network().ConnsToPeer(unreachable2.ID())
//	for _, conn := range conns {
//		fmt.Printf("连接类型: %s\n", conn.RemoteMultiaddr().String())
//	}
//
//	// 设置客户端2接收自定义协议消息的处理程序
//	unreachable2.SetStreamHandler("/customprotocol", func(s network.Stream) {
//		// 读取消息内容
//		buf := make([]byte, 1024)
//		n, err := s.Read(buf)
//		if err != nil {
//			log.Printf("读取消息失败: %v\n", err)
//			s.Close()
//			return
//		}
//		log.Printf("客户端2收到来自 %s 的消息: %s\n", s.Conn().RemotePeer(), string(buf[:n]))
//		s.Close()
//	})
//
//	// 服务端退出
//	fmt.Println("服务端即将退出...")
//	relay1.Close()
//	fmt.Println("服务端已退出")
//
//	// 等待2秒，开始客户端之间的通信测试
//	time.Sleep(2 * time.Second)
//
//	// 测试客户端间的直接通信
//	fmt.Println("开始测试客户端间的直接通信...")
//	testDirectCommunication(unreachable1, unreachable2)
//}
//
//func createClient(name string, relay peer.AddrInfo) (host.Host, error) {
//	h, err := libp2p.New(
//		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
//		libp2p.EnableRelay(),
//		libp2p.EnableHolePunching(),
//		libp2p.ForceReachabilityPrivate(),
//	)
//	if err != nil {
//		return nil, err
//	}
//	fmt.Printf("%s的ID是: %s，使用的端口: %d\n", name, h.ID(), getPortFromAddrs(h.Addrs()))
//	return h, nil
//}
//
//func testDirectCommunication(client1, client2 host.Host) {
//	success := make(chan bool)
//	messageCount := 0
//	maxMessages := 5
//
//	go func() {
//		for i := 0; i < maxMessages; i++ {
//			s, err := client1.NewStream(context.Background(), client2.ID(), "/customprotocol")
//			if err != nil {
//				fmt.Printf("测试 %d: 创建流失败: %v\n", i+1, err)
//				time.Sleep(time.Second)
//				continue
//			}
//			message := fmt.Sprintf("测试消息 %d", i+1)
//			_, err = s.Write([]byte(message))
//			if err != nil {
//				fmt.Printf("测试 %d: 发送消息失败: %v\n", i+1, err)
//				s.Close()
//				continue
//			}
//			fmt.Printf("测试 %d: 成功发送消息: %s\n", i+1, message)
//			s.Close()
//			messageCount++
//			time.Sleep(time.Second) // 每次发送间隔1秒
//		}
//		success <- messageCount == maxMessages
//	}()
//
//	select {
//	case result := <-success:
//		if result {
//			fmt.Printf("客户端间通信测试完成！成功发送 %d 条消息\n", messageCount)
//		} else {
//			fmt.Printf("客户端间通信不完整，只发送了 %d/%d 条消息\n", messageCount, maxMessages)
//		}
//	case <-time.After(10 * time.Second):
//		fmt.Printf("测试超时，只发送了 %d/%d 条消息\n", messageCount, maxMessages)
//	}
//}
//
//// 从Multiaddr中提取端口号
//func getPortFromAddrs(addrs []ma.Multiaddr) int {
//	for _, addr := range addrs {
//		// 解析地址，匹配端口号
//		var port int
//		fmt.Sscanf(addr.String(), "/ip4/127.0.0.1/tcp/%d", &port)
//		if port > 0 {
//			return port
//		}
//	}
//	return 0
//}

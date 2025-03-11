package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

type PeerInfo struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Addrs []string `json:"addrs"`
}

var (
	serverAddr = flag.String("server", "", "服务端地址 (格式: /ip4/127.0.0.1/tcp/9000/p2p/QmXXX)")
	clientName = flag.String("name", "", "客户端名称")
)

func main() {
	flag.Parse()

	if *serverAddr == "" || *clientName == "" {
		log.Fatal("请提供服务端地址和客户端名称")
	}

	// 创建客户端节点
	client, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
		libp2p.EnableRelay(),
		libp2p.EnableHolePunching(),
		libp2p.ForceReachabilityPrivate(),
	)
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	fmt.Printf("客户端 [%s] ID: %s\n", *clientName, client.ID())
	fmt.Println("客户端地址:")
	for _, addr := range client.Addrs() {
		fmt.Printf("%s/p2p/%s\n", addr, client.ID())
	}

	// 跟踪已连接的对等点
	connectedPeers := &sync.Map{}

	// 设置消息处理程序
	client.SetStreamHandler("/chat/1.0.0", func(s network.Stream) {
		defer s.Close()
		buf := make([]byte, 1024)
		n, err := s.Read(buf)
		if err != nil {
			return
		}
		msg := string(buf[:n])
		fmt.Printf("\n收到来自 [%s] 的消息: %s\n> ", getPeerName(connectedPeers, s.Conn().RemotePeer()), msg)
	})

	// 解析服务端地址
	serverMA, err := multiaddr.NewMultiaddr(*serverAddr)
	if err != nil {
		log.Fatalf("解析服务端地址失败: %v", err)
	}

	serverAddrInfo, err := peer.AddrInfoFromP2pAddr(serverMA)
	if err != nil {
		log.Fatalf("获取服务端信息失败: %v", err)
	}

	// 连接到服务端
	fmt.Println("正在连接服务端...")
	if err := client.Connect(context.Background(), *serverAddrInfo); err != nil {
		log.Fatalf("连接服务端失败: %v", err)
	}
	fmt.Println("成功连接到服务端")

	// 注册到服务端并获取其他客户端列表
	stream, err := client.NewStream(context.Background(), serverAddrInfo.ID, "/register/1.0.0")
	if err != nil {
		log.Fatalf("注册失败: %v", err)
	}

	// 发送客户端名称
	stream.Write([]byte(*clientName))

	// 接收其他客户端列表
	buf := make([]byte, 4096)
	n, err := stream.Read(buf)
	if err != nil {
		log.Fatalf("读取对等点列表失败: %v", err)
	}
	stream.Close()

	// 解析对等点列表
	var peers []*PeerInfo
	if err := json.Unmarshal(buf[:n], &peers); err != nil {
		log.Fatalf("解析对等点列表失败: %v", err)
	}

	// 连接到其他客户端
	for _, p := range peers {
		peerID, err := peer.Decode(p.ID)
		if err != nil {
			continue
		}

		addrs := make([]multiaddr.Multiaddr, 0)
		for _, addr := range p.Addrs {
			ma, err := multiaddr.NewMultiaddr(addr)
			if err != nil {
				continue
			}
			addrs = append(addrs, ma)
		}

		// 存储对等点信息
		connectedPeers.Store(peerID, p)
		fmt.Printf("已存储对等点 [%s] ID: %s, 地址: %v\n", p.Name, p.ID, p.Addrs)

		// 连接到对等点
		if err := client.Connect(context.Background(), peer.AddrInfo{
			ID:    peerID,
			Addrs: addrs,
		}); err != nil {
			fmt.Printf("连接到客户端 [%s] 失败: %v\n", p.Name, err)
			continue
		}
		fmt.Printf("成功连接到客户端 [%s]\n", p.Name)
	}

	// 处理用户输入
	fmt.Println("\n开始聊天（输入消息后按回车发送，输入 'quit' 退出）:")
	fmt.Print("> ")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		msg := scanner.Text()
		if msg == "quit" {
			break
		}

		if msg == "" {
			fmt.Print("> ")
			continue
		}

		// 发送消息给所有连接的对等点
		fmt.Println("开始发送消息给已连接的对等点...")
		var sentCount int
		connectedPeers.Range(func(key, value interface{}) bool {
			peerID := key.(peer.ID)
			peerInfo := value.(*PeerInfo)

			stream, err := client.NewStream(context.Background(), peerID, "/chat/1.0.0")
			if err != nil {
				fmt.Printf("创建到 [%s] 的流失败: %v\n", peerInfo.Name, err)
				return true
			}
			defer stream.Close()

			if _, err := stream.Write([]byte(msg)); err != nil {
				fmt.Printf("发送消息到 [%s] 失败: %v\n", peerInfo.Name, err)
				return true
			}
			fmt.Printf("成功向 [%s] 发送消息\n", peerInfo.Name)
			sentCount++
			return true
		})

		fmt.Println("消息发送完成")
		if sentCount == 0 {
			fmt.Println("未连接其他客户端，尝试重新请求对等点列表...")
			stream, err := client.NewStream(context.Background(), serverAddrInfo.ID, "/register/1.0.0")
			if err != nil {
				log.Printf("重新请求对等点列表失败: %v\n", err)
			} else {
				// 发送客户端名称
				stream.Write([]byte(*clientName))
				
				buf := make([]byte, 4096)
				n, err := stream.Read(buf)
				if err != nil {
					log.Printf("读取对等点列表失败: %v\n", err)
				} else {
					var newPeers []*PeerInfo
					if err := json.Unmarshal(buf[:n], &newPeers); err != nil {
						log.Printf("解析对等点列表失败: %v\n", err)
					} else {
						for _, p := range newPeers {
							peerID, err := peer.Decode(p.ID)
							if err != nil {
								continue
							}
							addrs := make([]multiaddr.Multiaddr, 0)
							for _, addr := range p.Addrs {
								ma, err := multiaddr.NewMultiaddr(addr)
								if err != nil {
									continue
								}
								addrs = append(addrs, ma)
							}
							connectedPeers.Store(peerID, p)
							if err := client.Connect(context.Background(), peer.AddrInfo{
								ID:    peerID,
								Addrs: addrs,
							}); err != nil {
								log.Printf("重新连接到客户端 [%s] 失败: %v\n", p.Name, err)
							} else {
								log.Printf("重新连接到客户端 [%s] 成功\n", p.Name)
							}
						}
					}
				}
				stream.Close()
			}
		}
		fmt.Printf("消息已发送给 %d 个客户端\n", sentCount)
		fmt.Print("> ")
	}
}

func getPeerName(peers *sync.Map, id peer.ID) string {
	if value, ok := peers.Load(id); ok {
		return value.(*PeerInfo).Name
	}
	return id.String()[:12]
}

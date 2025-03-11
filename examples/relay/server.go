package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	"github.com/libp2p/go-libp2p/p2p/protocol/holepunch"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"
	ma "github.com/multiformats/go-multiaddr"
)

type PeerInfo struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Addrs []string `json:"addrs"`
}

func main() {
	// 创建服务端节点
	fmt.Println("开始创建服务端...")
	server, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/9000"),
		libp2p.EnableRelayService(),
		libp2p.EnableHolePunching(),
	)
	if err != nil {
		log.Fatalf("创建服务端失败: %v", err)
	}
	defer server.Close()

	// 打印服务端信息
	fmt.Printf("服务端ID: %s\n", server.ID().String())
	fmt.Println("服务端地址:")
	for _, addr := range server.Addrs() {
		fmt.Printf("%s/p2p/%s\n", addr.String(), server.ID().String())
	}

	// 配置中继服务
	_, err = relay.New(server)
	if err != nil {
		log.Fatalf("配置中继服务失败: %v", err)
	}

	// 配置identify服务
	ids, err := identify.NewIDService(server)
	if err != nil {
		log.Fatalf("配置identify服务失败: %v", err)
	}

	// 配置打洞服务
	_, err = holepunch.NewService(server, ids, func() []ma.Multiaddr { return server.Addrs() })
	if err != nil {
		log.Fatalf("配置打洞服务失败: %v", err)
	}

	// 跟踪连接的客户端
	var (
		connectedPeers = make(map[peer.ID]*PeerInfo)
		mu            sync.Mutex
	)

	// 设置客户端注册处理程序
	server.SetStreamHandler("/register/1.0.0", func(s network.Stream) {
		defer s.Close()
		
		// 读取客户端名称
		buf := make([]byte, 1024)
		n, err := s.Read(buf)
		if err != nil {
			return
		}
		clientName := string(buf[:n])
		
		peerID := s.Conn().RemotePeer()
		
		mu.Lock()
		// 添加新客户端
		peerInfo := &PeerInfo{
			ID:    peerID.String(),
			Name:  clientName,
			Addrs: make([]string, 0),
		}
		// 获取客户端的地址
		peerInfo.Addrs = append(peerInfo.Addrs, s.Conn().RemoteMultiaddr().String())
		connectedPeers[peerID] = peerInfo
		
		// 准备其他客户端列表
		otherPeers := make([]*PeerInfo, 0)
		for id, info := range connectedPeers {
			if id != peerID {
				otherPeers = append(otherPeers, info)
			}
		}
		mu.Unlock()

		// 发送其他客户端列表给新客户端
		peerList, _ := json.Marshal(otherPeers)
		s.Write(peerList)
		
		fmt.Printf("新客户端 [%s] 已注册，当前连接数: %d\n", clientName, len(connectedPeers))
	})

	// 监听连接断开事件
	server.Network().Notify(&network.NotifyBundle{
		DisconnectedF: func(n network.Network, conn network.Conn) {
			mu.Lock()
			defer mu.Unlock()
			
			peerID := conn.RemotePeer()
			if info, exists := connectedPeers[peerID]; exists {
				fmt.Printf("客户端 [%s] 断开连接，当前连接数: %d\n", info.Name, len(connectedPeers)-1)
				delete(connectedPeers, peerID)
			}
		},
	})

	// 保持服务端运行
	select {}
} 
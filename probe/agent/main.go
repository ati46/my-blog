package main

import (
	"flag"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// 定义配置变量（通过命令行标志接收）
var (
	serverDomain string
	serverPort   string
	mySecret     string
	useSSL       bool
)

// AgentPayload 定义上报的数据包格式
// 注意：只上报原始网络计数 (RawNet)，不计算速度，速度由服务端计算
type AgentPayload struct {
	HostName  string  `json:"host_name"`
	CPU       float64 `json:"cpu"`
	MemUsed   float64 `json:"mem_used"`
	MemTotal  float64 `json:"mem_total"`
	DiskUsed  uint64  `json:"disk_used"`
	RawNetIn  uint64  `json:"raw_net_in"`  // 原始下载字节总数
	RawNetOut uint64  `json:"raw_net_out"` // 原始上传字节总数
}

func main() {
	// 1. 解析命令行参数
	// 默认值：域名 localhost, 端口 8080, 不开启 SSL
	flag.StringVar(&mySecret, "s", "", "必填: 本机的唯一通信密钥 (Secret)")
	flag.StringVar(&serverDomain, "d", "localhost", "服务端域名或IP")
	flag.StringVar(&serverPort, "p", "8080", "服务端端口")
	flag.BoolVar(&useSSL, "ssl", false, "是否开启 SSL/TLS (如果你使用了 Cloudflare 或 Nginx 反代，请开启此项)")
	
	flag.Parse()

	// 2. 校验必填项
	if mySecret == "" {
		log.Fatal("❌ 启动失败: 未指定 Secret。\n👉 用法示例: ./agent -s \"my_key_123\" -d \"example.com\" -p \"443\" -ssl")
	}

	// 3. 构建连接 URL
	scheme := "ws"
	if useSSL {
		scheme = "wss"
	}
	u := url.URL{
		Scheme:   scheme,
		Host:     serverDomain + ":" + serverPort,
		Path:     "/ws",
		RawQuery: "secret=" + mySecret,
	}

	log.Printf("🚀 探针启动中...")
	log.Printf("📡 目标服务端: %s", u.String())
	log.Printf("🔑 使用密钥: %s", mySecret)

	// 4. 主循环：负责连接和断线重连
	for {
		// 尝试连接
		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			log.Printf("❌ 连接失败: %v (5秒后重试...)", err)
			time.Sleep(5 * time.Second)
			continue
		}
		log.Println("✅ 成功连接到服务端，开始采集数据...")

		// 5. 内层循环：负责采集和发送数据
		for {
			// --- 数据采集 (耗时约1秒) ---
			
			// CPU: 阻塞等待 1 秒来计算使用率 (这也充当了上报间隔的心跳)
			cPercent, _ := cpu.Percent(time.Second, false)
			
			// 基础信息
			hInfo, _ := host.Info()
			mInfo, _ := mem.VirtualMemory()
			dInfo, _ := disk.Usage("/")
			nInfo, _ := net.IOCounters(false) // 获取所有网卡的总流量

			// 数据清洗
			cpuVal := 0.0
			if len(cPercent) > 0 {
				cpuVal = cPercent[0]
			}

			var rIn, rOut uint64
			if len(nInfo) > 0 {
				rIn = nInfo[0].BytesRecv
				rOut = nInfo[0].BytesSent
			}

			// 组装数据包
			data := AgentPayload{
				HostName:  hInfo.Hostname,
				CPU:       cpuVal,
				MemUsed:   float64(mInfo.Used) / 1024 / 1024 / 1024, // 转为 GB
				MemTotal:  float64(mInfo.Total) / 1024 / 1024 / 1024, // 转为 GB
				DiskUsed:  dInfo.Used / 1024 / 1024 / 1024,          // 转为 GB
				RawNetIn:  rIn,  // 原始字节，交给服务端去算速度
				RawNetOut: rOut, // 原始字节
			}

			// --- 发送数据 ---
			err := c.WriteJSON(data)
			if err != nil {
				log.Println("⚠️ 发送失败，连接可能已断开:", err)
				c.Close() // 关闭旧连接
				break     // 跳出内层循环，触发外层重连
			}
		}
	}
}

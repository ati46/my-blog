package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// ProbeData 定义上报的数据包结构
type ProbeData struct {
	HostName  string  `json:"host_name"`
	OS        string  `json:"os"`
	Uptime    uint64  `json:"uptime"`
	CPU       float64 `json:"cpu"`        // CPU 使用率 %
	MemUsed   uint64  `json:"mem_used"`   // 内存已用 MB
	MemTotal  uint64  `json:"mem_total"`  // 内存总计 MB
	DiskUsed  uint64  `json:"disk_used"`  // 硬盘已用 GB
	DiskTotal uint64  `json:"disk_total"` // 硬盘总计 GB
	NetIn     uint64  `json:"net_in"`     // 入网流量 Total Bytes
	NetOut    uint64  `json:"net_out"`    // 出网流量 Total Bytes
}

func collectData() (*ProbeData, error) {
	// 1. 主机基本信息
	hInfo, _ := host.Info()
	
	// 2. CPU 使用率 (获取 1 秒内的平均值)
	cPercent, _ := cpu.Percent(time.Second, false)
	
	// 3. 内存信息
	mInfo, _ := mem.VirtualMemory()
	
	// 4. 磁盘信息 (根路径)
	dInfo, _ := disk.Usage("/")
	
	// 5. 网络流量 (获取所有网卡的总和)
	nInfo, _ := net.IOCounters(false)

	data := &ProbeData{
		HostName:  hInfo.Hostname,
		OS:        hInfo.Platform + " " + hInfo.PlatformVersion,
		Uptime:    hInfo.Uptime,
		CPU:       0, // 默认值，防止panic
		MemUsed:   mInfo.Used / 1024 / 1024,
		MemTotal:  mInfo.Total / 1024 / 1024,
		DiskUsed:  dInfo.Used / 1024 / 1024 / 1024,
		DiskTotal: dInfo.Total / 1024 / 1024 / 1024,
		NetIn:     0,
		NetOut:    0,
	}

	if len(cPercent) > 0 {
		data.CPU = cPercent[0]
	}
	if len(nInfo) > 0 {
		data.NetIn = nInfo[0].BytesRecv
		data.NetOut = nInfo[0].BytesSent
	}

	return data, nil
}

func main() {
	fmt.Println("🚀 G-Probe Agent 启动中...")

	// 模拟每秒上报一次
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		data, err := collectData()
		if err != nil {
			log.Println("采集错误:", err)
			continue
		}

		// 序列化为 JSON
		jsonData, _ := json.MarshalIndent(data, "", "  ")
		
		// 这里未来会替换为 WebSocket 发送代码
		fmt.Printf("📊 实时数据快照:\n%s\n------------------\n", string(jsonData))
	}
}

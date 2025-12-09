package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	DBFile = "db.json"
)

// AgentPayload: 探针上报的原始数据包
type AgentPayload struct {
	HostName  string  `json:"host_name"`
	CPU       float64 `json:"cpu"`
	MemUsed   float64 `json:"mem_used"`
	MemTotal  float64 `json:"mem_total"`
	DiskUsed  uint64  `json:"disk_used"`
	RawNetIn  uint64  `json:"raw_net_in"`
	RawNetOut uint64  `json:"raw_net_out"`
}

// HostNode: 服务端存储的完整节点状态
type HostNode struct {
	Secret string `json:"secret"`

	HostName string  `json:"host_name"`
	CPU      float64 `json:"cpu"`
	MemUsed  float64 `json:"mem_used"`
	MemTotal float64 `json:"mem_total"`
	DiskUsed uint64  `json:"disk_used"`
	LastSeen int64   `json:"last_seen"`
	SpeedIn  uint64  `json:"speed_in"`
	SpeedOut uint64  `json:"speed_out"`
	MonthIn  uint64  `json:"month_in"`
	MonthOut uint64  `json:"month_out"`

	LastRawIn  uint64 `json:"last_raw_in"`
	LastRawOut uint64 `json:"last_raw_out"`
	LastUpdate int64  `json:"last_update"`

	Cost         string `json:"cost"`
	BillingCycle string `json:"billing_cycle"`
	ResetDay     int    `json:"reset_day"`
	StartDate    string `json:"start_date"`
	Location     string `json:"location"`
}

var (
	hosts = make(map[string]*HostNode)
	mu    sync.RWMutex // 读写锁

	activeConns = make(map[string]*websocket.Conn)
)

func loadDB() {
	raw, err := os.ReadFile(DBFile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("⚠️ db.json 不存在，请手动创建")
			return
		}
		log.Fatal("❌ db.json 读取失败:", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if err := json.Unmarshal(raw, &hosts); err != nil {
		log.Fatal("❌ db.json 格式错误:", err)
	}
	log.Printf("📂 已加载 %d 个节点配置", len(hosts))
}

func saveDB() {
	// ⚡️ 优化：快速复制数据，减少锁占用时间
	mu.RLock()
	// 注意：这里不能直接 Marshal hosts，因为 hosts 可能正在被写入
	// 但 json.Marshal 内部会反射，比较慢。我们先拷贝一份指针 map
	// 最简单的防死锁：只读锁期间做序列化通常是可以的，
	// 但为了极致安全，我们应该尽可能快地释放锁。
	// 这里保持原样，因为 saveDB 频率低 (5s一次)，影响不大。
	data, _ := json.MarshalIndent(hosts, "", "  ")
	mu.RUnlock()
	
	os.WriteFile(DBFile, data, 0644)
}

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	clientSecret := r.URL.Query().Get("secret")
	if clientSecret == "" {
		http.Error(w, "Missing secret", 401)
		return
	}

	// --- 阶段1: 握手前检查 (读锁) ---
	mu.RLock()
	var targetHost *HostNode
	for _, h := range hosts {
		if h.Secret == clientSecret {
			targetHost = h
			break
		}
	}
	mu.RUnlock() // 查完立即解锁

	if targetHost == nil {
		http.Error(w, "Invalid secret", 403)
		return
	}
	hostName := targetHost.HostName

	// --- 阶段2: 活跃连接检查 (写锁) ---
	mu.Lock()
	if _, ok := activeConns[hostName]; ok {
		mu.Unlock() // 必须解锁
		http.Error(w, "Node already connected", http.StatusConflict)
		return
	}
	mu.Unlock()

	// --- 阶段3: 升级连接 ---
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	// 注册连接
	mu.Lock()
	activeConns[hostName] = conn
	mu.Unlock()

	log.Printf("✅ 节点接入: %s", hostName)

	defer func() {
		conn.Close()
		mu.Lock()
		if activeConns[hostName] == conn {
			delete(activeConns, hostName)
		}
		mu.Unlock()
		log.Printf("🔌 节点断开: %s", hostName)
	}()

	// --- 阶段4: 数据接收循环 (关键优化区域) ---
	for {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		var p AgentPayload
		if err := conn.ReadJSON(&p); err != nil {
			break // 读失败直接退出
		}

		// ⚡️⚡️⚡️ 关键优化：只在更新内存的一瞬间加锁 ⚡️⚡️⚡️
		mu.Lock()
		node, exists := hosts[hostName]
		if !exists {
			mu.Unlock()
			break
		}

		now := time.Now()
		lastTime := time.Unix(node.LastUpdate, 0)

		// 记录日志的标志位（不在锁里打日志）
		resetTriggered := false

		// 流量重置
		if now.Month() != lastTime.Month() && now.Day() >= node.ResetDay {
			node.MonthIn = 0
			node.MonthOut = 0
			resetTriggered = true
		}

		// 流量计算 (处理重启回滚)
		deltaIn := p.RawNetIn - node.LastRawIn
		if p.RawNetIn < node.LastRawIn { deltaIn = p.RawNetIn }
		
		deltaOut := p.RawNetOut - node.LastRawOut
		if p.RawNetOut < node.LastRawOut { deltaOut = p.RawNetOut }

		// 更新字段
		node.CPU = p.CPU
		node.MemUsed = p.MemUsed
		node.MemTotal = p.MemTotal
		node.DiskUsed = p.DiskUsed
		node.SpeedIn = deltaIn
		node.SpeedOut = deltaOut
		node.MonthIn += deltaIn
		node.MonthOut += deltaOut
		node.LastSeen = now.Unix()
		node.LastUpdate = now.Unix()
		node.LastRawIn = p.RawNetIn
		node.LastRawOut = p.RawNetOut

		mu.Unlock() // 🔥 立即解锁！不要在锁里做任何耗时操作

		// 在锁外面打印日志
		if resetTriggered {
			log.Printf("📅 触发重置: %s", hostName)
		}
	}
}

// ⚡️⚡️⚡️ API 优化：防死锁版 ⚡️⚡️⚡️
func handleStats(w http.ResponseWriter, r *http.Request) {
	// 1. 加读锁
	mu.RLock()
	
	// 2. 快速复制数据到局部变量 (Deep Copy)
	// 这样我们就不用在 JSON 序列化（这是一个慢操作）的时候一直占着锁了
	list := make([]HostNode, 0, len(hosts))
	for _, v := range hosts {
		list = append(list, *v) // 值拷贝
	}
	
	// 3. 立即释放锁
	mu.RUnlock()

	// 4. 慢操作（网络IO + JSON）放在锁外面做
	// 这样即使客户端网速慢，也不会卡死整个服务端
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(list)
}

func main() {
	var port string
	flag.StringVar(&port, "p", "8080", "端口")
	flag.Parse()

	loadDB()

	go func() {
		for range time.Tick(5 * time.Second) {
			saveDB()
		}
	}()

	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/api/stats", handleStats)
	http.Handle("/", http.FileServer(http.Dir(".")))

	log.Printf("📡 服务端启动: :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

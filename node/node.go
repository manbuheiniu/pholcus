﻿package node

import (
	"github.com/henrylee2cn/pholcus/node/crawlpool"
	"github.com/henrylee2cn/pholcus/node/spiderqueue"
	"github.com/henrylee2cn/pholcus/node/task"
	"github.com/henrylee2cn/pholcus/runtime/cache"
	"github.com/henrylee2cn/pholcus/runtime/status"
	"github.com/henrylee2cn/teleport"
	"log"
	"strconv"
	"time"
)

type Node struct {
	// 运行模式
	RunMode int
	// 服务器端口号
	Port string
	// 服务器地址（不含Port）
	Master string
	// socket长连接双工通信接口，json数据传输
	teleport.Teleport
	// 节点间传递的任务的存储库
	*task.TaskJar
	// 当前任务的蜘蛛队列
	Spiders spiderqueue.SpiderQueue
	// 爬行动作的回收池
	Crawls crawlpool.CrawlPool
	// 节点状态
	Status int
}

func NewNode(mode int, port int, master string) *Node {
	return &Node{
		RunMode:  mode,
		Port:     ":" + strconv.Itoa(port),
		Master:   master,
		Teleport: teleport.New(),
		TaskJar:  task.NewTaskJar(),
		Spiders:  spiderqueue.New(),
		Crawls:   crawlpool.New(),
		Status:   status.RUN,
	}
}

// 初始化并运行节点
func (self *Node) Run() {
	switch self.RunMode {
	case status.SERVER:
		if self.checkPort() {
			log.Printf("                                                                                               ！！当前运行模式为：[ 服务器 ] 模式！！")
			self.Teleport.SetAPI(ServerApi(self)).Server(self.Port)
		}

	case status.CLIENT:
		if self.checkAll() {
			log.Printf("                                                                                               ！！当前运行模式为：[ 客户端 ] 模式！！")
			self.Teleport.SetAPI(ClientApi(self)).Client(self.Master, self.Port)
		}
	case status.OFFLINE:
		log.Printf("                                                                                               ！！当前运行模式为：[ 单机 ] 模式！！")
		return
	default:
		log.Println(" *    ——请指定正确的运行模式！——")
		return
	}
	// 开启实时log发送
	go self.log()
}

func (self *Node) Stop() {
	self.Teleport.Close()
	self.Status = status.STOP
}

// 返回节点数
func (self *Node) CountNodes() int {
	return self.Teleport.CountNodes()
}

// 生成task并添加至库，服务器模式专用
func (self *Node) AddNewTask() (tasksNum, spidersNum int) {
	length := self.Spiders.Len()
	t := task.Task{}

	// 从配置读取字段
	t.ThreadNum = cache.Task.ThreadNum
	t.Pausetime = cache.Task.Pausetime
	t.OutType = cache.Task.OutType
	t.DockerCap = cache.Task.DockerCap
	t.DockerQueueCap = cache.Task.DockerQueueCap
	t.MaxPage = cache.Task.MaxPage

	for i, sp := range self.Spiders.GetAll() {

		t.Spiders = append(t.Spiders, map[string]string{"name": sp.GetName(), "keyword": sp.GetKeyword()})
		spidersNum++

		// 每十个蜘蛛存为一个任务
		if i > 0 && i%10 == 0 && length > 10 {
			// 存入
			one := t
			self.TaskJar.Push(&one)
			// log.Printf(" *     [新增任务]   详情： %#v", *t)

			tasksNum++

			// 清空spider
			t.Spiders = []map[string]string{}
		}
	}

	if len(t.Spiders) != 0 {
		// 存入
		one := t
		self.TaskJar.Push(&one)
		// log.Printf(" *     [新增任务]   详情： %#v", *t)
		tasksNum++
	}
	return
}

// 客户端请求获取任务
func (self *Node) GetTaskAlways() {
	self.Request(nil, "task")
}

// 客户端模式下获取任务
func (self *Node) DownTask() *task.Task {
ReStartLabel:
	for self.CountNodes() == 0 {
		if len(self.TaskJar.Tasks) != 0 {
			break
		}
		time.Sleep(5e7)
	}

	if len(self.TaskJar.Tasks) == 0 {
		self.GetTaskAlways()
		for len(self.TaskJar.Tasks) == 0 {
			if self.CountNodes() == 0 {
				goto ReStartLabel
			}
			time.Sleep(5e7)
		}
	}
	return self.TaskJar.Pull()
}

func (self *Node) log() {
	for {
		if self.Status == status.STOP {
			return
		}
		self.Teleport.Request(<-cache.SendChan, "log")
	}
}

func (self *Node) checkPort() bool {
	if cache.Task.Port == 0 {
		log.Println(" *     —— 亲，分布式端口不能为空哦~")
		return false
	}
	return true
}

func (self *Node) checkAll() bool {
	if cache.Task.Master == "" || !self.checkPort() {
		log.Println(" *     —— 亲，服务器地址不能为空哦~")
		return false
	}
	return true
}

﻿package distribute

import (
	"github.com/henrylee2cn/teleport"
	"log"
)

func ServerApi(n subApp) teleport.API {
	return teleport.API{
		// 提供任务给客户端
		"task": &task1{n},

		// 打印接收到的报告
		"log": new(log1),
	}
}

type task1 struct {
	subApp
}

func (self *task1) Process(receive *teleport.NetData) *teleport.NetData {
	return teleport.ReturnData(self.Out(self.CountNodes()))
}

type log1 struct{}

func (*log1) Process(receive *teleport.NetData) *teleport.NetData {
	log.Printf(" * ")
	log.Printf(" *     [ %s ]    %s", receive.From, receive.Body)
	log.Printf(" * ")
	return nil
}

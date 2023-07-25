package main

import (
	"net"
	"strings"
)

type User struct {
	Name string
	Addr string
	C    chan string
	conn net.Conn

	server *Server
}

//创建一个用户API
func NewUser(conn net.Conn, server *Server) *User {
	userAddr := conn.RemoteAddr().String()
	user := &User{
		Name:   userAddr,
		Addr:   userAddr,
		C:      make(chan string),
		conn:   conn,
		server: server,
	}
	//启动监听当前新用户的channel的goroutine
	go user.ListenMessage()

	return user
}

//用户的上线业务
func (this *User) Online() {
	//将用户加入表
	this.server.mapLock.Lock()
	this.server.OnlineMap[this.Name] = this
	this.server.mapLock.Unlock()

	//将用户登录消息加入广播channel
	this.server.BroadCast(this, "已上线")
}

//用户的下线业务
func (this *User) Offline() {
	//将用户从OnlineMap表中删除
	this.server.mapLock.Lock()
	delete(this.server.OnlineMap, this.Name)
	this.server.mapLock.Unlock()

	//将用户下线消息加进行广播
	this.server.BroadCast(this, "下线")
}

//客户端写消息
func (this *User) SendMsg(msg string) {
	this.conn.Write([]byte(msg))
}

//用户处理消息的业务
func (this *User) DoMessage(msg string) {
	if msg == "who" {
		//查询当前用户都有哪些
		this.server.mapLock.Lock()
		for _, user := range this.server.OnlineMap {
			onLineMsg := "[" + user.Addr + "]" + user.Name + ":" + "在线...\n"
			this.SendMsg(onLineMsg)
		}
		this.server.mapLock.Unlock()
	} else if len(msg) > 7 && msg[:7] == "rename|" { //重命名功能
		newName := strings.Split(msg, "|")[1]

		_, ok := this.server.OnlineMap[newName]
		if ok {
			this.SendMsg("当前用户名已存在\n")
		} else {
			this.server.mapLock.Lock()
			delete(this.server.OnlineMap, this.Name)
			this.server.OnlineMap[newName] = this
			this.server.mapLock.Unlock()

			this.Name = newName
			this.SendMsg("您已更新用户名：" + this.Name + "\n")
		}
	} else if len(msg) > 4 && msg[:3] == "to|" { //私聊功能
		//私聊功能
		//消息模型：“to|张三|...”
		remoteName := strings.Split(msg, "|")[1]
		if remoteName == "" {
			this.SendMsg("消息格式不正确，请使用\"to|张三|你好啊\"格式,\n")
			return
		}

		toMessage := strings.Split(msg, "|")[2]
		if toMessage == "" {
			this.SendMsg("无消息内容，请重发\n")
			return
		}

		remoteUser, ok := this.server.OnlineMap[remoteName]
		if ok {
			//找到了
			remoteUser.SendMsg(this.Name + "对您说:" + toMessage + "\n")
			this.SendMsg("发送成功\n")
		} else {
			//没找到用户
			this.SendMsg("找不到该用户\n")
			return
		}
	} else {
		this.server.BroadCast(this, msg)
	}
}

//监听当前User channel,一有消息就发送给对端客户端
func (this *User) ListenMessage() {
	for {
		msg := <-this.C

		this.conn.Write([]byte(msg + "\n"))
	}
}

package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Server struct {
	Ip   string
	Port int

	//在线用户列表
	OnlineMap map[string]*User
	mapLock   sync.RWMutex

	//消息广播channel
	Message chan string
}

func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip:        ip,
		Port:      port,
		OnlineMap: make(map[string]*User),
		Message:   make(chan string),
	}

	return server
}

//监听Message的消息，一旦有消息则进行广播
func (this *Server) ListenMessage() {
	for {
		msg := <-this.Message

		//广播给在线的用户
		this.mapLock.Lock()
		for _, cli := range this.OnlineMap {
			cli.C <- msg
		}
		this.mapLock.Unlock()
	}
}

//将用户信息广播
func (this *Server) BroadCast(user *User, msg string) {
	//将消息加入到channel
	sendMsg := "[" + user.Addr + "]" + user.Name + ":" + msg

	this.Message <- sendMsg
}

func (this *Server) Handler(conn net.Conn) {

	user := NewUser(conn, this)

	user.Online()

	//监听用户是否活跃的channel
	isLive := make(chan bool)

	//接收客户端发送的信息
	go func() {
		buf := make([]byte, 2096)
		for {
			n, err := conn.Read(buf)
			if n == 0 {
				user.Offline()
				return
			}

			if err != nil && err != io.EOF {
				fmt.Println("Conn Read err:", err)
				return
			}
			//提取用户信息，跳过行尾的换行符
			msg := string(buf[:n-1])

			//针对用户信息处理
			user.DoMessage(msg)

			//用户任意消息代表用户活跃
			isLive <- true
		}
	}()

	for {
		select {
		case <-isLive: //不做任何事情，在活跃的时候能激活定时器，重置时间
			//一旦不活跃则不激活定时器，让其一直计时直到时间到触发事件
		case <-time.After(time.Second * 1000):
			//超时关闭资源，conn,handle,C
			user.SendMsg("你被踢了")

			close(user.C)

			conn.Close()

			return
		}
	}

}

func (this *Server) Start() {
	//socket listen
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", this.Ip, this.Port))
	if err != nil {
		fmt.Println("net.Listen err", err)
		return
	}

	//close listen socket
	defer listener.Close()

	//启动监听，一旦开启服务器就开始监听
	go this.ListenMessage()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("listener accept err:", err)
			continue
		}

		//do handler
		go this.Handler(conn)
	}
}

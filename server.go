package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Server struct {
	Ip        string
	Port      int
	OnlineMap map[string]*User // online user map
	mapLock   sync.RWMutex     // lock
	Message   chan string
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

// listening on Message broadcast channel's goroutine.
// once any user is online, send it to all users
func (this *Server) ListenMessage() {
	for {
		msg := <-this.Message

		this.mapLock.Lock()
		for _, cli := range this.OnlineMap {
			cli.C <- msg
		}
		this.mapLock.Unlock()
	}
}

func (this *Server) Handler(conn net.Conn) {
	// user login
	user := NewUser(conn, this)
	user.Online()

	//listen on active users
	isLive := make(chan bool)

	// receive message from client side
	go func() {
		buf := make([]byte, 4096)

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

			msg := string(buf[:n-1])

			user.DoMessage(msg)
			isLive <- true
		}
	}()

	for {
		select {
		case <-isLive:

		case <-time.After(time.Second * 10):
			// over time
			user.SendMessage("Your connection has been closed!\n")
			close(user.C)
			conn.Close()
			return
		}
	}
}

func (this *Server) BroadCast(user *User, msg string) {
	sendMsg := "[" + user.Addr + "]" + user.Name + " : " + msg
	this.Message <- sendMsg
}

// server start
func (this *Server) Start() {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", this.Ip, this.Port))
	if err != nil {
		fmt.Println("Listener error: ", err)
		return

	}

	//close listen socket
	defer listener.Close()

	// message listening
	go this.ListenMessage()

	for {
		//accept
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("listener accept err:", err)
			continue
		}

		//do handler
		go this.Handler(conn)
	}
}

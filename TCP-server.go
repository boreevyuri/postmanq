package main

import (
	"fmt"
	"net"
)

func main() {
	fmt.Println("Launching server...")

	// Listen port
	ln, _ := net.Listen("tcp", ":1025")

	// Let's open port
	conn, _ := ln.Accept()

	// Launch cycle
	//for {
	// Messages must have \n at the end
	//message, _ := bufio.NewReader(conn).ReadString('\n')
	// Print request
	//fmt.Print("Message received: ", string(message))
	// generate response
	newmessage := "421 This IP is in blacklist. Write to abuse@corp.mail.ru"
	_, _ = conn.Write([]byte(newmessage + "\n"))
	//}
}

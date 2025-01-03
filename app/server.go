package main

import (
	"fmt"
	"net"
	"os"
	"slices"
	"strings"
)

func main() {
	fmt.Println("Server starting")

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	accepted_protocols := []string{"HTTP/1.0", "HTTP/1.1"}
	accepted_methods := []string{"GET", "PUT", "DELETE"}
	const ok = "HTTP/1.1 200 OK\r\n\r\n"
	const not_found = "HTTP/1.1 404 Not Found\r\n\r\n"
	buffer := make([]byte, 1024)

	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading: ", err.Error())
		return
	}

	req := strings.Fields(string(buffer[:n]))
	method := req[0]
	path := req[1]
	protocol := req[2]

	if !slices.Contains(accepted_protocols, protocol) {
		fmt.Println("HTTP Version Not Supported", protocol)
		conn.Write([]byte("HTTP/1.1 505 HTTP Version Not Supported\r\n\r\n"))
		return
	}

	if !slices.Contains(accepted_methods, method) {
		fmt.Println("Method Not Allowed", method)
		conn.Write([]byte("HTTP/1.1 405 Method Not Allowed\r\n\r\n"))
		return
	}

	fmt.Println("path", strings.Split(path, "/"))
	sub_strings := strings.Split(path, "/")[1:] // cut first off because its empty space, I guess Split "/" cuts and makes the left side empty

	for _, v := range sub_strings {
		fmt.Printf("printing %s \n", v)
	}

	fmt.Println(sub_strings[0])
	if path == "/" {
		conn.Write([]byte(ok))
	} else if len(sub_strings) > 0 && sub_strings[0] == "echo" {
		payload := sub_strings[1]
		content_length := fmt.Sprintf("%v", len(payload))
		ok := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: " + content_length + "\r\n\r\n" + payload

		conn.Write([]byte(ok))
		return // is this return necessary?
	} else {
		conn.Write([]byte(not_found))
	}
}

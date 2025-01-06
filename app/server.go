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

func parseRequest(buffer []byte, n int) [][]string {
	fmt.Println("START -- Parsing request buffer")
	var result [][]string

	req_buffer_string := string(buffer[:n])
	// fmt.Println(req_buffer_string)
	req_chunks := strings.Split(req_buffer_string, "\n")

	for i, c := range req_chunks {
		line := strings.TrimSpace(c)
		if line == "" {
			continue
		}

		if i == 0 {
			// we are in the very fist line, which is request method
			fmt.Println("req", line)
			parts := strings.Split(line, " ")
			fmt.Println("parts", parts)
			result = append(result, []string{parts[0], parts[1], parts[2]})
		} else if i > 0 {
			// headers
			parts := strings.Split(line, ":")
			key := strings.ToLower(strings.TrimSpace(parts[0]))
			value := strings.ToLower(strings.TrimSpace(parts[1]))
			result = append(result, []string{key, value})
		}

	}
	fmt.Println("DONE -- Parsing request buffer")
	return result
}

func parsePath(p string) []string {
	var result []string
	if p == "/" {
		result = append(result, "/")
	} else {
		paths := strings.Split(p, "/")
		for _, c := range paths {
			result = append(result, c)
		}
	}

	fmt.Println("paths result", result)
	return result
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

	parsed_req := parseRequest(buffer, n)
	fmt.Println(parsed_req)
	method := parsed_req[0][0]
	path := parsePath(parsed_req[0][1]) //this can be "/" or subpaths too like "/abc/bde/cfg"
	protocol := parsed_req[0][2]

	// DEBUG STUFF
	fmt.Println("pased_req", parsed_req)
	fmt.Println("method", method)
	fmt.Println("path", path)
	fmt.Println("protocol", protocol)

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

	if path[0] == "/" {
		conn.Write([]byte(ok))
	} else if path[1] == "echo" {
		content_length := fmt.Sprintf("%v", len(path[2]))
		ok := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: " + content_length + "\r\n\r\n" + path[2]
		conn.Write([]byte(ok))
	} else if path[1] == "user-agent" {
		fmt.Println("in user agent")
		// find user agent header in request
		headers := parsed_req[1:]
		idx := slices.IndexFunc(headers, func(s []string) bool {
			return strings.Contains(s[0], "user-agent")
		})

		if idx != -1 {
			user_agent := headers[idx][1]
			content_length := fmt.Sprintf("%v", len(user_agent))
			message_body := user_agent
			ok := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: " + content_length + "\r\n\r\n" + message_body
			conn.Write([]byte(ok))
		} else {
			fmt.Println("user-agent header not found")
		}

	} else {
		conn.Write([]byte(not_found))
	}
}

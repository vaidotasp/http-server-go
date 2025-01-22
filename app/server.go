package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type Config struct {
	Directory         string
	AcceptedProtocols []string
	AcceptedMethods   []string
	AcceptedEncoding  []string
}

var globalConfig Config

func setupConfig() {
	var dirFlag = flag.String("directory", "", "help msg for my dir flag")
	flag.Parse()

	if *dirFlag == "" {
		fmt.Println("Error: no dir argument provided, provide --directory when running program")
	}

	globalConfig = Config{
		Directory:         *dirFlag,
		AcceptedProtocols: []string{"HTTP/1.0", "HTTP/1.1"},
		AcceptedMethods:   []string{"GET", "PUT", "DELETE", "POST"},
		AcceptedEncoding:  []string{"gzip"},
	}
}

func findFileInDir(filename string) (found bool, size int64, content string, err error) {
	// make sure its abosule path
	file_path, err := filepath.Abs(globalConfig.Directory)
	if err != nil {
		fmt.Println("Error reading: ", err.Error())
		return found, size, content, err
	}
	// read dir
	c, err := os.ReadDir(file_path)
	if err != nil {
		fmt.Println("No dir found, need to create: ", err.Error())
		err = os.MkdirAll(file_path, 0755)
		return found, size, content, err
	}

	fmt.Println("Listing subdir/parent")
	fmt.Println(filename)
	for _, entry := range c {
		fmt.Println(" ", entry.Name(), entry.IsDir())
		// if is not dir and name matches, we found it!
		if !entry.IsDir() && entry.Name() == filename {
			found = true
			file_absolute_path := file_path + "/" + filename
			fmt.Println(file_absolute_path)

			f, err := os.ReadFile(file_absolute_path)
			if err != nil {
				fmt.Println("err opening file")
				os.Exit(1)
			}
			content = string(f)

			info, err := os.Stat(file_absolute_path)
			fmt.Println("info", info.Size())
			size = info.Size()
			break
		}
	}
	return found, size, content, err
}

func createOrUpdateFileInDir(filename string, body string) (err error) {
	// make sure its abosule path
	dir_path, err := filepath.Abs(globalConfig.Directory)
	if err != nil {
		fmt.Println("Error reading: ", err.Error())
		return err
	}
	// read dir
	err = os.MkdirAll(dir_path, 0755)
	if err != nil {
		fmt.Println("No dir found, need to create: ", err.Error())
		return err
	}

	fmt.Println("Listing subdir/parent")
	fmt.Println(filename)

	err = os.WriteFile(dir_path+"/"+filename, []byte(body), 0644)
	if err != nil {
		panic(err)
	}
	return err
}

func checkClientEncoding(parts [][]string) (encoding string, err error) {
	// find in parts encoding header
	idx := slices.IndexFunc(parts, func(s []string) bool {
		return strings.Contains(s[0], "accept-encoding")
	})

	if idx == -1 {
		// client not accepting any compression
		return "", fmt.Errorf("accept-encoding header not found")
	}

	client_encoding := parts[idx][1]
	fmt.Println(client_encoding)
	client_encodings := strings.Split(client_encoding, ",")
	fmt.Println(client_encodings)
	var accepted_filtered_encodings []string

	if len(client_encodings) == 0 {
		return "none", err
	}

	for _, e := range globalConfig.AcceptedEncoding {
		fmt.Println("going over:", e)
		fmt.Println("comparing:", client_encodings, e)
		// this is not supporting additional params like  (gzip;q=1.0)
		for _, ce := range client_encodings {
			trimmed := strings.TrimSpace(ce)
			if strings.EqualFold(trimmed, e) {
				accepted_filtered_encodings = append(accepted_filtered_encodings, e)
			}
		}
	}

	fmt.Println(accepted_filtered_encodings)

	if len(accepted_filtered_encodings) == 0 {
		return "none", err
	}

	// we are gonna just support one encoding for now, the first one in line, and we add those in terms of preference
	return accepted_filtered_encodings[0], err
}

func main() {
	fmt.Println("Server started")

	setupConfig()

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
	req_chunks := strings.Split(req_buffer_string, "\n")

	for i, c := range req_chunks {
		line := strings.TrimSpace(c)
		fmt.Println("debug - i:", i)
		if i == 0 {
			// we are in the very fist line, which is request method, url and protocol
			fmt.Println("req", line)
			parts := strings.Split(line, " ")
			fmt.Println("parts", parts)
			result = append(result, []string{parts[0], parts[1], parts[2]})
			fmt.Println("debug - res:", result)
		} else if i > 0 {
			// headers
			if line == "" {
				// we have a data in POST most likely here because empty line separates headers from data body
				fmt.Println("debug - body:", result)
				parts := strings.Split(line, ":")
				result = append(result, []string{parts[0]})
			} else {
				if result[i-1][0] == "" { // prev entry is empty string, we are in "body" section
					parts := strings.Split(line, ":")
					result = append(result, []string{parts[0]})
				} else {
					parts := strings.Split(line, ":")
					fmt.Println("parts", parts)
					key := strings.ToLower(strings.TrimSpace(parts[0]))
					value := strings.ToLower(strings.TrimSpace(parts[1]))
					result = append(result, []string{key, value})
					fmt.Println("debug - res:", result)
				}
			}
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
	const ok = "HTTP/1.1 200 OK\r\n\r\n"
	const ok_created = "HTTP/1.1 201 Created\r\n\r\n"
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

	if !slices.Contains(globalConfig.AcceptedProtocols, protocol) {
		fmt.Println("HTTP Version Not Supported", protocol)
		conn.Write([]byte("HTTP/1.1 505 HTTP Version Not Supported\r\n\r\n"))
		return
	}

	if !slices.Contains(globalConfig.AcceptedMethods, method) {
		fmt.Println("Method Not Allowed", method)
		conn.Write([]byte("HTTP/1.1 405 Method Not Allowed\r\n\r\n"))
		return
	}

	// check if client accepts any ecoding and if so which one. compare to what we support on the server.
	//  return can be "none" or "gzip",etc
	encoding, err := checkClientEncoding(parsed_req)

	if path[0] == "/" {
		conn.Write([]byte(ok))
	} else if path[1] == "echo" {
		content_length := fmt.Sprintf("%v", len(path[2]))
		ok := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: " + content_length + "\r\n\r\n" + path[2]

		fmt.Println(encoding)
		if encoding != "none" {
			ok = "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Encoding: " + encoding + "\r\n" + "Content-Length: " + content_length + "\r\n\r\n" + path[2]
		}
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
	} else if path[1] == "files" {
		if method == "GET" {
			file_name := path[2]
			file_found, size, content, err := findFileInDir(file_name)
			if err != nil {
				fmt.Println("ERROR: file not found in dir")
			}
			if file_found {
				fmt.Printf("found, %v, size: %v \n", file_found, size)
				s := fmt.Sprintf("%v", size)
				ok := "HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: " + s + "\r\n\r\n" + content
				conn.Write([]byte(ok))
			} else {
				conn.Write([]byte(not_found))
			}
		} else if method == "POST" {
			file_name := path[2]
			fmt.Println("we here?")
			body := parsed_req[len(parsed_req)-1][0]
			fmt.Println("body", body)
			err := createOrUpdateFileInDir(file_name, parsed_req[len(parsed_req)-1][0])
			if err != nil {
				fmt.Println("ERROR: file not found in dir")
				conn.Write([]byte(not_found))
			}
			conn.Write([]byte(ok_created))
		}
	} else {
		conn.Write([]byte(not_found))
	}
}

package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func startServer(t *testing.T) func() {

	testDir := t.TempDir()
	oldArgs := os.Args

	os.Args = []string{
		oldArgs[0],
		"--directory", testDir,
	}

	go func() {
		main()
	}()

	time.Sleep(500 * time.Millisecond)

	return func() {
		os.Args = oldArgs
	}
}

func TestMainPath(t *testing.T) {
	cleanup := startServer(t)
	defer cleanup()

	t.Run("Root path", func(t *testing.T) {
		conn, err := net.Dial("tcp", "127.0.0.1:4221")
		if err != nil {
			t.Fatalf("Failed to connect to the server: %v", err)
		}
		defer conn.Close()
		request := "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n"
		if _, err := conn.Write([]byte(request)); err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}

		buffer := make([]byte, 1024)
		n, err := conn.Read((buffer))
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}
		response := string(buffer[:n])
		if !strings.Contains(response, "HTTP/1.1 200 OK") {
			t.Errorf("Expected 'HTTP/1.1 200 OK', got:\n%s", response)
		}
	})

	t.Run("404 handling all unknown paths", func(t *testing.T) {
		conn, err := net.Dial("tcp", "127.0.0.1:4221")
		if err != nil {
			t.Fatalf("Failed to connect to the server: %v", err)
		}
		defer conn.Close()
		request := "GET /abc HTTP/1.1\r\nHost: localhost\r\n\r\n"
		if _, err := conn.Write([]byte(request)); err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}

		buffer := make([]byte, 1024)
		n, err := conn.Read((buffer))
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}
		response := string(buffer[:n])
		expected_response := "HTTP/1.1 404 Not Found"
		if !strings.Contains(response, expected_response) {
			t.Errorf("Expected '%s', got:\n%s", expected_response, response)
		}
	})
	t.Run("/echo/abc path - no encoding", func(t *testing.T) {
		/* Given /echo path we test:
		take everything that is after echo/ so /echo/abc, we take abc string and send it back
		eg. 	ok := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: " + content_length + "\r\n\r\n" + path[2]

		this test does not assume encoding, in fact we do not provide encoding in our request so return must not be encoded
		*/
		conn, err := net.Dial("tcp", "127.0.0.1:4221")
		if err != nil {
			t.Fatalf("Failed to connect to the server: %v", err)
		}
		defer conn.Close()
		request := "GET /echo/abc HTTP/1.1\r\nHost: localhost:4221\r\n\r\n"
		if _, err := conn.Write([]byte(request)); err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}

		buffer := make([]byte, 1024)
		n, err := conn.Read((buffer))
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}
		content_length := fmt.Sprintf("%v", len("abc"))
		response := string(buffer[:n])
		expected_response := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: " + content_length + "\r\n\r\n" + "abc"

		if response != expected_response {
			t.Errorf("Expected equality, but got response \n%s\n != expected_response \n%s\n", response, expected_response)
		}
	})
	t.Run("/echo/ path - no encoding", func(t *testing.T) {
		/* Given /echo path we test:
		- no subpath after echo is present, expected behavior is 422 Unprocessable Content
		*/
		conn, err := net.Dial("tcp", "127.0.0.1:4221")
		if err != nil {
			t.Fatalf("Failed to connect to the server: %v", err)
		}
		defer conn.Close()
		request := "GET /echo/ HTTP/1.1\r\nHost: localhost:4221\r\n\r\n"
		if _, err := conn.Write([]byte(request)); err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}

		buffer := make([]byte, 1024)
		n, err := conn.Read((buffer))
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}
		response := string(buffer[:n])
		expected_response := "HTTP/1.1 422 Unprocessable Entity"

		if response != expected_response {
			t.Errorf("Expected equality, but got response \n%s\n != expected_response \n%s\n", response, expected_response)
		}
	})
	t.Run("/echo/abc/cde path - no encoding", func(t *testing.T) {
		/* Given /echo/abc/cde path we test:
		- subpath is present, further path is ignored
		*/
		conn, err := net.Dial("tcp", "127.0.0.1:4221")
		if err != nil {
			t.Fatalf("Failed to connect to the server: %v", err)
		}
		defer conn.Close()
		request := "GET /echo/abc/cde HTTP/1.1\r\nHost: localhost:4221\r\n\r\n"
		if _, err := conn.Write([]byte(request)); err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}

		buffer := make([]byte, 1024)
		n, err := conn.Read((buffer))
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}
		content_length := fmt.Sprintf("%v", len("abc"))
		response := string(buffer[:n])
		expected_response := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: " + content_length + "\r\n\r\n" + "abc"

		if response != expected_response {
			t.Errorf("Expected equality, but got response \n%s\n != expected_response \n%s\n", response, expected_response)
		}
	})
	t.Run("echo path - gzip encoding", func(t *testing.T) {
		// test gzip encoding, gzip "abc" (path after /echo) and send it back
		conn, err := net.Dial("tcp", "127.0.0.1:4221")
		if err != nil {
			t.Fatalf("Failed to connect to the server: %v", err)
		}
		defer conn.Close()
		request := "GET /echo/abc HTTP/1.1\r\nHost: localhost:4221\r\nAccept-Encoding: gzip\r\n\r\n"
		if _, err := conn.Write([]byte(request)); err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}

		buffer := make([]byte, 1024)
		n, err := conn.Read((buffer))
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}
		var b bytes.Buffer
		str := "abc"
		gz := gzip.NewWriter(&b)
		gz.Write([]byte(str))
		gz.Close()

		output := b.String()
		length := strconv.Itoa(b.Len())
		compression := "Content-Encoding: gzip\r\n"
		response := string(buffer[:n])
		expected_response := "HTTP/1.1 200 OK\r\n" +
			compression +
			"Content-Type: " + "text/plain" + "\r\n" +
			"Content-Length: " + length +
			"\r\n" +
			"\r\n" + output

		if response != expected_response {
			t.Errorf("Expected equality, but got response \n%s\n != expected_response \n%s\n", response, expected_response)
		}
	})
	t.Run("echo path - brotli encoding -- not found", func(t *testing.T) {
		// test not supported encoding (like brotli), dont encode in that case, just send it back as is
		conn, err := net.Dial("tcp", "127.0.0.1:4221")
		if err != nil {
			t.Fatalf("Failed to connect to the server: %v", err)
		}
		defer conn.Close()
		request := "GET /echo/abc HTTP/1.1\r\nHost: localhost:4221\r\nAccept-Encoding: br\r\n\r\n"
		if _, err := conn.Write([]byte(request)); err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}

		buffer := make([]byte, 1024)
		n, err := conn.Read((buffer))
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}
		str := "abc"
		length := fmt.Sprintf("%v", len("abc"))
		response := string(buffer[:n])
		expected_response := "HTTP/1.1 200 OK\r\n" +
			"Content-Type: " + "text/plain" + "\r\n" +
			"Content-Length: " + length +
			"\r\n" +
			"\r\n" + str

		if response != expected_response {
			t.Errorf("Expected equality, but got response \n%s\n != expected_response \n%s\n", response, expected_response)
		}
	})
	t.Run("TODO: test /user-agent", func(t *testing.T) {
		// testing reading User-Agent header and spitting it back out as a response body
		header_val := "foobar/1.2.3"
		conn, err := net.Dial("tcp", "127.0.0.1:4221")
		if err != nil {
			t.Fatalf("Failed to connect to the server: %v", err)
		}
		defer conn.Close()
		request := "GET /user-agent HTTP/1.1\r\nHost: localhost:4221\r\nUser-Agent: " + header_val + "\r\n\r\n"
		if _, err := conn.Write([]byte(request)); err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}

		buffer := make([]byte, 1024)
		n, err := conn.Read((buffer))
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}
		content_length := fmt.Sprintf("%v", len(header_val))
		response := string(buffer[:n])
		expected_response := "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: " + content_length + "\r\n\r\n" + header_val

		if response != expected_response {
			t.Errorf("Expected equality, but got response \n%s\n != expected_response \n%s\n", response, expected_response)
		}
	})
	t.Run("TODO: test /files -> GET", func(t *testing.T) {})
	t.Run("test /files -> POST", func(t *testing.T) {
		header_val := "foobar/1.2.3"
		conn, err := net.Dial("tcp", "127.0.0.1:4221")
		if err != nil {
			t.Fatalf("Failed to connect to the server: %v", err)
		}
		defer conn.Close()

		request := "POST /files/number HTTP/1.1\r\nHost: localhost:4221\r\n" + "Content-Type: application/octet-stream\r\n" + "User-Agent: " + header_val + "\r\n\r\n" + "1234"

		if _, err := conn.Write([]byte(request)); err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}

		buffer := make([]byte, 1024)
		n, err := conn.Read((buffer))
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}
		response := string(buffer[:n])
		expected_response := "HTTP/1.1 201 Created\r\n\r\n"
		if response != expected_response {
			t.Errorf("Expected equality, but got response \n%s\n != expected_response \n%s\n", response, expected_response)
		}
	})

}

package main

import (
	"fmt"
	"net"
	"os"
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
	t.Run("echo path - encoding", func(t *testing.T) {})
	t.Run("404 handling all unknown paths", func(t *testing.T) {})
	t.Run("404 handling all unknown paths", func(t *testing.T) {})
}

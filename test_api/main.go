package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	crypt "crypto/tls"

	tls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
)

func main() {
	token := os.Getenv("ISHOSTING_API_TOKEN")
	if token == "" {
		fmt.Println("ISHOSTING_API_TOKEN environment variable is not set")
		os.Exit(1)
	}

	dialTLS := func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialer := &net.Dialer{Timeout: 30 * time.Second}
		conn, err := dialer.DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}

		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}

		tlsConn := tls.UClient(conn, &tls.Config{ServerName: host}, tls.HelloChrome_Auto)
		if err := tlsConn.Handshake(); err != nil {
			conn.Close()
			return nil, err
		}
		return tlsConn, nil
	}

	// Use HTTP/2 transport since the server negotiates h2 via ALPN
	h2Transport := &http2.Transport{
		DialTLSContext: func(ctx context.Context, network, addr string, _ *crypt.Config) (net.Conn, error) {
			return dialTLS(ctx, network, addr)
		},
	}

	client := &http.Client{
		Transport: h2Transport,
		Timeout:   30 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, "https://api.ishosting.com/settings/ssh", nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		os.Exit(1)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Api-Token", token)
	req.Header.Set("Accept-Language", "en")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")

	fmt.Println("Sending request to https://api.ishosting.com/profile")
	fmt.Println("Using uTLS with Chrome fingerprint + HTTP/2\n")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Proto: %s\n", resp.Proto)
	fmt.Printf("Response Headers: %v\n\n", resp.Header)
	fmt.Printf("Body:\n%s\n", string(body))
}

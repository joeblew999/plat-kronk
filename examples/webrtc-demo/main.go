// WebRTC multi-party video conference using Livekit.
// Server generates tokens, Livekit handles all WebRTC complexity.
package main

import (
	"bufio"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/livekit/protocol/auth"
	"github.com/skip2/go-qrcode"
)

//go:embed index.html static/*
var content embed.FS

// Livekit configuration - defaults for dev mode
// Override via environment variables for production:
//   LIVEKIT_API_KEY, LIVEKIT_API_SECRET, LIVEKIT_HOST
// Must match livekit/livekit.yaml
var (
	livekitAPIKey    = getEnvOrDefault("LIVEKIT_API_KEY", "devkey")
	livekitAPISecret = getEnvOrDefault("LIVEKIT_API_SECRET", "n2Z2GBJV8l6Jggu85GWnID5LRmRThuYjgy2VLV8GGnKPqs26")
	livekitHost      = getEnvOrDefault("LIVEKIT_HOST", "ws://localhost:7880")
)

// getEnvOrDefault returns the environment variable value or the default
func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

var (
	tunnelURL string
	tunnelMu  sync.RWMutex

	// Room passwords (room name -> password). Empty password means no auth required.
	roomPasswords = map[string]string{
		"demo-room": "",         // No password for default room
		"private":   "secret123", // Example protected room
	}
)

func main() {
	addr := flag.String("addr", ":9080", "HTTP listen address")
	noOpen := flag.Bool("no-open", false, "Don't open browser automatically")
	flag.Parse()

	// Find an available port
	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	localURL := fmt.Sprintf("http://localhost:%d", port)

	// Get LAN IP for mobile access
	lanIP := getLanIP()
	lanURL := ""
	if lanIP != "" {
		lanURL = fmt.Sprintf("http://%s:%d", lanIP, port)
	}

	fmt.Println("===========================================")
	fmt.Println("  Livekit Video Conference")
	fmt.Println("===========================================")
	fmt.Println()
	fmt.Println("Access URLs:")
	fmt.Printf("  Local:  %s\n", localURL)
	if lanURL != "" {
		fmt.Printf("  LAN:    %s  (for mobile/other devices)\n", lanURL)
	}
	fmt.Println()
	fmt.Println("Livekit server: http://localhost:7880")
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	http.HandleFunc("/", serveIndex)
	http.HandleFunc("/token", handleToken)
	http.HandleFunc("/tunnel-url", handleTunnelURL)
	http.HandleFunc("/qr.png", handleQRCode)
	http.HandleFunc("/livekit-url", handleLivekitURL)
	http.HandleFunc("/static/", serveStatic)

	// Watch cloudflared log for tunnel URL
	go watchTunnelURL()

	if !*noOpen {
		go func() {
			log.Println("Waiting for tunnel URL (HTTPS required for iOS)...")
			for i := 0; i < 30; i++ {
				tunnelMu.RLock()
				url := tunnelURL
				tunnelMu.RUnlock()
				if url != "" {
					log.Printf("Opening browser to HTTPS: %s", url)
					openBrowser(url)
					return
				}
				time.Sleep(2 * time.Second)
			}
			log.Println("Tunnel URL not detected, opening local HTTP URL")
			openBrowser(localURL)
		}()
	}

	log.Fatal(http.Serve(listener, nil))
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	log.Printf("HTTP request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
	data, _ := content.ReadFile("index.html")
	w.Header().Set("Content-Type", "text/html")
	w.Write(data)
}

func serveStatic(w http.ResponseWriter, r *http.Request) {
	// Strip leading slash to match embed path
	path := strings.TrimPrefix(r.URL.Path, "/")
	data, err := content.ReadFile(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Set content type based on extension
	if strings.HasSuffix(path, ".js") {
		w.Header().Set("Content-Type", "application/javascript")
	} else if strings.HasSuffix(path, ".css") {
		w.Header().Set("Content-Type", "text/css")
	}
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(data)
}

// handleToken generates a Livekit access token for a participant
func handleToken(w http.ResponseWriter, r *http.Request) {
	room := r.URL.Query().Get("room")
	identity := r.URL.Query().Get("identity")
	password := r.URL.Query().Get("password")

	if room == "" {
		room = "demo-room"
	}
	if identity == "" {
		identity = fmt.Sprintf("user-%d", time.Now().UnixNano()%10000)
	}

	// Check room password
	if expectedPass, exists := roomPasswords[room]; exists && expectedPass != "" {
		if password != expectedPass {
			http.Error(w, "Invalid password", http.StatusUnauthorized)
			return
		}
	}

	// Create token with permissions
	at := auth.NewAccessToken(livekitAPIKey, livekitAPISecret)
	grant := &auth.VideoGrant{
		RoomJoin: true,
		Room:     room,
	}
	at.AddGrant(grant).SetIdentity(identity).SetValidFor(24 * time.Hour)

	token, err := at.ToJWT()
	if err != nil {
		http.Error(w, "Token generation failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token":    token,
		"room":     room,
		"identity": identity,
	})
}

// handleLivekitURL returns the livekit WebSocket URL
// Returns localhost when accessed locally, tunnel URL when accessed remotely
func handleLivekitURL(w http.ResponseWriter, r *http.Request) {
	tunnelMu.RLock()
	tURL := tunnelURL
	tunnelMu.RUnlock()

	// Default to local LiveKit
	livekitURL := livekitHost

	// If accessed via tunnel domain (not localhost), use tunnel for LiveKit too
	host := r.Host
	if tURL != "" && !strings.Contains(host, "localhost") && !strings.Contains(host, "127.0.0.1") {
		// Use tunnel for livekit WebSocket - Caddy proxies /rtc to LiveKit
		livekitURL = strings.Replace(tURL, "https://", "wss://", 1)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": livekitURL})
}

func handleTunnelURL(w http.ResponseWriter, r *http.Request) {
	tunnelMu.RLock()
	url := tunnelURL
	tunnelMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	if url == "" {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"url":""}`))
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"url": url})
}

func handleQRCode(w http.ResponseWriter, r *http.Request) {
	tunnelMu.RLock()
	url := tunnelURL
	tunnelMu.RUnlock()

	if url == "" {
		http.Error(w, "No tunnel URL", http.StatusNotFound)
		return
	}

	png, err := qrcode.Encode(url, qrcode.Medium, 256)
	if err != nil {
		http.Error(w, "QR generation failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(png)
}

func watchTunnelURL() {
	// Default to cloudflared/.data/cloudflared.log, can be overridden via CF_LOG env var
	logFile := getEnvOrDefault("CF_LOG", "../../cloudflared/.data/cloudflared.log")
	urlPattern := regexp.MustCompile(`https://[a-z0-9-]+\.trycloudflare\.com`)

	for {
		data, err := os.ReadFile(logFile)
		if err == nil {
			scanner := bufio.NewScanner(strings.NewReader(string(data)))
			for scanner.Scan() {
				line := scanner.Text()
				if match := urlPattern.FindString(line); match != "" {
					tunnelMu.Lock()
					if tunnelURL != match {
						tunnelURL = match
						log.Printf("Tunnel URL detected: %s", match)
					}
					tunnelMu.Unlock()
					break
				}
			}
		}
		time.Sleep(2 * time.Second)
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}
	if cmd != nil {
		cmd.Start()
	}
}

func getLanIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() || ip.To4() == nil {
				continue
			}

			return ip.String()
		}
	}
	return ""
}

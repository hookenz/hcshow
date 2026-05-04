package handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"net"
	"net/http"

	go_qr "github.com/piglig/go-qr"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	pbtpl "github.com/pocketbase/pocketbase/tools/template"
)

func PrintJudgeCard(registry *pbtpl.Registry) func(*core.RequestEvent) error {
	ip, err := getLocalIP()
	if err != nil {
		ip = "localhost"
	}

	judgeURL := fmt.Sprintf("http://%s:8090/judge", ip)
	fmt.Printf("Judge card URL: %s\n", judgeURL)

	qrSVG, err := generateQRSVG(judgeURL)
	if err != nil {
		// Non-fatal — page will just show without QR
		fmt.Printf("Warning: could not generate QR code: %v\n", err)
		qrSVG = ""
	}

	return func(e *core.RequestEvent) error {
		html, err := registry.LoadFiles(
			"views/layout.html",
			"views/judge.html",
		).Render(map[string]any{
			"localIP":  ip,
			"judgeURL": judgeURL,
			"qrSVG":    qrSVG,
		})
		if err != nil {
			return e.InternalServerError("", err)
		}
		return e.HTML(http.StatusOK, html)
	}
}

func ShowScanner(app *pocketbase.PocketBase, registry *pbtpl.Registry) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		html, err := registry.LoadFiles(
			"views/layout.html",
			"views/scanner.html",
		).Render(nil)
		if err != nil {
			return e.InternalServerError("", err)
		}
		return e.HTML(http.StatusOK, html)
	}
}

func generateQRSVG(content string) (template.HTML, error) {
	qr, err := go_qr.EncodeText(content, go_qr.Medium)
	if err != nil {
		return "", err
	}

	config := go_qr.NewQrCodeImgConfig(8, 4)

	var buf bytes.Buffer
	if err := qr.WriteAsSVG(config, &buf, "#FFFFFF", "#000000"); err != nil {
		return "", err
	}

	return template.HTML(buf.String()), nil
}

// func getLocalIP() (string, error) {
// 	ifaces, err := net.Interfaces()
// 	if err != nil {
// 		return "", err
// 	}
// 	for _, iface := range ifaces {
// 		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
// 			continue
// 		}
// 		addrs, err := iface.Addrs()
// 		if err != nil {
// 			continue
// 		}
// 		for _, addr := range addrs {
// 			var ip net.IP
// 			switch v := addr.(type) {
// 			case *net.IPNet:
// 				ip = v.IP
// 			case *net.IPAddr:
// 				ip = v.IP
// 			}
// 			if ip == nil || ip.IsLoopback() {
// 				continue
// 			}
// 			if ip = ip.To4(); ip != nil {
// 				return ip.String(), nil
// 			}
// 		}
// 	}
// 	return "", fmt.Errorf("no local IP found")
// }

func getLocalIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

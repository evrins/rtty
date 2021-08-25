package utils

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func OpenBrowser(url string) {
	args := []string{}
	switch runtime.GOOS {
	case "windows":
		r := strings.NewReplacer("&", "^&")
		args = []string{"cmd", "start", "/", r.Replace(url)}
	case "linux":
		args = []string{"xdg-open", url}
	case "darwin":
		args = []string{"open", url}
	}

	//nolint
	out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
	if err != nil {
		log.Printf("%s: %s\n", out, err)
	}
}

func Filter(ss []string) []string {
	rs := []string{}

	for _, s := range ss {
		if s == "" {
			continue
		}
		rs = append(rs, s)
	}

	return rs
}

func GetEnv(key, def string) string {
	v := os.Getenv(key)
	if v != "" {
		return v
	}
	return def
}

func WindowSize(msg interface{}) (rows, cols uint16, err error) {
	data, ok := msg.(map[string]interface{})
	if !ok {
		return 0, 0, fmt.Errorf("invalid message: %#+v", msg)
	}

	rows = uint16(data["rows"].(float64))
	cols = uint16(data["cols"].(float64))

	return
}

func Register(consulHost string, port int) (err error) {
	client, err := api.NewClient(&api.Config{
		Address: consulHost,
		Scheme:  "http",
	})
	if err != nil {
		return
	}
	ip, err := GetIP()
	if err != nil {
		return
	}
	serviceId := "rtty-" + ip
	err = client.Agent().ServiceRegister(&api.AgentServiceRegistration{
		ID:      serviceId,
		Name:    serviceId,
		Tags:    []string{"rtty"},
		Port:    port,
		Address: ip,
		Check: &api.AgentServiceCheck{
			Interval:                       "5s",
			Timeout:                        "5s",
			TTL:                            "",
			HTTP:                           fmt.Sprintf("http://%s:%d/live", ip, port),
			DeregisterCriticalServiceAfter: "5s",
		},
	})
	if err != nil {
		return
	}
	return
}

func GetIP() (ip string, err error) {
	conn, err := net.Dial("tcp", "192.168.224.81:53")
	if err != nil {
		return
	}
	defer conn.Close()
	addr := conn.LocalAddr().String()
	ip = strings.Split(addr, ":")[0]
	return
}

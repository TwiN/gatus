package zeroconf

import (
	"net"
	"sync"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/logr"
	"github.com/hashicorp/mdns"
)

const (
	instanceName = "gatus"
	hostName     = "gatus.local."
)

var (
	server *mdns.Server
	mutex  sync.Mutex
)

func Initialize(cfg *config.Config) {
	mutex.Lock()
	defer mutex.Unlock()
	service, err := mdns.NewMDNSService(instanceName, "_http._tcp", "", hostName, cfg.Web.Port, localIPs(), []string{"path=/"})
	if err != nil {
		logr.Errorf("[zeroconf.Initialize] Failed to create mDNS service info: %s", err.Error())
		return
	}
	s, err := mdns.NewServer(&mdns.Config{Zone: service})
	if err != nil {
		logr.Errorf("[zeroconf.Initialize] Failed to start mDNS server: %s", err.Error())
		return
	}
	server = s
	logr.Infof("[zeroconf.Initialize] Advertising Gatus as '%s' (%s) on port %d via mDNS", instanceName, hostName, cfg.Web.Port)
}

func Shutdown() {
	mutex.Lock()
	defer mutex.Unlock()
	if server != nil {
		if err := server.Shutdown(); err != nil {
			logr.Errorf("[zeroconf.Shutdown] Failed to shut down mDNS server: %s", err.Error())
		}
		server = nil
		logr.Info("[zeroconf.Shutdown] Stopped advertising Gatus via mDNS")
	}
}

func localIPs() []net.IP {
	var ips []net.IP
	interfaces, err := net.Interfaces()
	if err != nil {
		logr.Warnf("[zeroconf.localIPs] Failed to list network interfaces, falling back to auto-detection: %s", err.Error())
		return nil
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		if iface.Name == "docker0" || len(iface.Name) > 3 && iface.Name[:3] == "br-" {
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

			if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}

			ips = append(ips, ip)
		}
	}
	return ips
}

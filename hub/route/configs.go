package route

import (
	"net/http"
	"path/filepath"

	"github.com/doreamon-design/clash/component/resolver"
	"github.com/doreamon-design/clash/config"
	C "github.com/doreamon-design/clash/constant"
	"github.com/doreamon-design/clash/hub/executor"
	"github.com/doreamon-design/clash/listener"
	"github.com/doreamon-design/clash/log"
	"github.com/doreamon-design/clash/tunnel"
	"github.com/go-zoox/zoox"

	"github.com/samber/lo"
)

func getConfigs(ctx *zoox.Context) {
	general := executor.GetGeneral()
	ctx.JSON(http.StatusOK, general)
}

func patchConfigs(ctx *zoox.Context) {
	general := struct {
		Port        *int               `json:"port"`
		SocksPort   *int               `json:"socks-port"`
		RedirPort   *int               `json:"redir-port"`
		TProxyPort  *int               `json:"tproxy-port"`
		MixedPort   *int               `json:"mixed-port"`
		AllowLan    *bool              `json:"allow-lan"`
		BindAddress *string            `json:"bind-address"`
		Mode        *tunnel.TunnelMode `json:"mode"`
		LogLevel    *log.LogLevel      `json:"log-level"`
		IPv6        *bool              `json:"ipv6"`
	}{}
	if err := ctx.BindJSON(&general); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrBadRequest)
		return
	}

	if general.Mode != nil {
		tunnel.SetMode(*general.Mode)
	}

	if general.LogLevel != nil {
		log.SetLevel(*general.LogLevel)
	}

	if general.IPv6 != nil {
		resolver.DisableIPv6 = !*general.IPv6
	}

	if general.AllowLan != nil {
		listener.SetAllowLan(*general.AllowLan)
	}

	if general.BindAddress != nil {
		listener.SetBindAddress(*general.BindAddress)
	}

	ports := listener.GetPorts()
	ports.Port = lo.FromPtrOr(general.Port, ports.Port)
	ports.SocksPort = lo.FromPtrOr(general.SocksPort, ports.SocksPort)
	ports.RedirPort = lo.FromPtrOr(general.RedirPort, ports.RedirPort)
	ports.TProxyPort = lo.FromPtrOr(general.TProxyPort, ports.TProxyPort)
	ports.MixedPort = lo.FromPtrOr(general.MixedPort, ports.MixedPort)

	listener.ReCreatePortsListeners(*ports, tunnel.TCPIn(), tunnel.UDPIn())

	ctx.Status(http.StatusNoContent)
}

func updateConfigs(ctx *zoox.Context) {
	req := struct {
		Path    string `json:"path"`
		Payload string `json:"payload"`
	}{}
	if err := ctx.BindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrBadRequest)
		return
	}

	force := ctx.Query().Get("force").Bool()
	var cfg *config.Config
	var err error

	if req.Payload != "" {
		cfg, err = executor.ParseWithBytes([]byte(req.Payload))
		if err != nil {
			ctx.JSON(http.StatusBadRequest, newError(err.Error()))
			return
		}
	} else {
		if req.Path == "" {
			req.Path = C.Path.Config()
		}
		if !filepath.IsAbs(req.Path) {
			ctx.JSON(http.StatusBadRequest, newError("path is not a absolute path"))
			return
		}

		cfg, err = executor.ParseWithPath(req.Path)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, newError(err.Error()))
			return
		}
	}

	executor.ApplyConfig(cfg, force)
	ctx.Status(http.StatusNoContent)
}

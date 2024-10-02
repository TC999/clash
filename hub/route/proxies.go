package route

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/doreamon-design/clash/adapter"
	"github.com/doreamon-design/clash/adapter/outboundgroup"
	"github.com/doreamon-design/clash/component/profile/cachefile"
	C "github.com/doreamon-design/clash/constant"
	"github.com/doreamon-design/clash/tunnel"
	"github.com/go-zoox/zoox"
)

func getProxyFromContext(ctx *zoox.Context) (C.Proxy, error) {
	name := ctx.Param().Get("name").String()
	proxies := tunnel.Proxies()
	if proxy, ok := proxies[name]; !ok {
		return nil, ErrNotFound
	} else {
		return proxy, nil
	}
}

func getProxies(ctx *zoox.Context) {
	proxies := tunnel.Proxies()
	ctx.JSON(http.StatusOK, zoox.H{
		"proxies": proxies,
	})
}

func getProxy(ctx *zoox.Context) {
	proxy, err := getProxyFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusNotFound, err)
		return
	}

	ctx.JSON(http.StatusOK, proxy)
}

func updateProxy(ctx *zoox.Context) {
	req := struct {
		Name string `json:"name"`
	}{}
	if err := ctx.BindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrBadRequest)
		return
	}

	proxy, err := getProxyFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusNotFound, err)
		return
	}

	proxy.Name()

	aProxy, ok := proxy.(*adapter.Proxy)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, newError("failed to transform proxy to *adapter.Proxy"))
		return
	}

	selector, ok := aProxy.ProxyAdapter.(*outboundgroup.Selector)
	if !ok {
		ctx.JSON(http.StatusBadRequest, newError("Must be a Selector"))
		return
	}

	if err := selector.Set(req.Name); err != nil {
		ctx.JSON(http.StatusBadRequest, newError(fmt.Sprintf("Selector update error: %s", err.Error())))
		return
	}

	cachefile.Cache().SetSelected(proxy.Name(), req.Name)
	ctx.Status(http.StatusNoContent)
}

func getProxyDelay(ctx *zoox.Context) {
	url := ctx.Query().Get("url").String()
	timeout := ctx.Query().Get("timeout").Int()

	proxy, err := getProxyFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusNotFound, err)
		return
	}

	ctx2, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(timeout))
	defer cancel()

	delay, meanDelay, err := proxy.URLTest(ctx2, url)
	if ctx2.Err() != nil {
		ctx.JSON(http.StatusGatewayTimeout, ErrRequestTimeout)
		return
	}

	if err != nil || delay == 0 {
		ctx.JSON(http.StatusServiceUnavailable, newError("An error occurred in the delay test"))
		return
	}

	ctx.JSON(http.StatusOK, zoox.H{
		"delay":     delay,
		"meanDelay": meanDelay,
	})
}

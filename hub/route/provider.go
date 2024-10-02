package route

import (
	"context"
	"net/http"
	"time"

	C "github.com/doreamon-design/clash/constant"
	"github.com/doreamon-design/clash/constant/provider"
	"github.com/doreamon-design/clash/tunnel"
	"github.com/go-zoox/zoox"

	"github.com/samber/lo"
)

func getProviders(ctx *zoox.Context) {
	providers := tunnel.Providers()
	ctx.JSON(http.StatusOK, zoox.H{
		"providers": providers,
	})
}

func getProvider(ctx *zoox.Context) {
	provider, err := getProviderFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusNotFound, err)
		return
	}

	ctx.JSON(http.StatusOK, provider)
}

func updateProvider(ctx *zoox.Context) {
	provider, err := getProviderFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusNotFound, err)
		return
	}

	if err := provider.Update(); err != nil {
		ctx.JSON(http.StatusServiceUnavailable, newError(err.Error()))
		return
	}

	ctx.Status(http.StatusNoContent)
}

func healthCheckProvider(ctx *zoox.Context) {
	provider, err := getProviderFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusNotFound, err)
		return
	}

	provider.HealthCheck()

	ctx.Status(http.StatusNoContent)
}

func getProxyFromProvider(ctx *zoox.Context) {
	proxy, err := getProxyInProviderFromContext(ctx)
	if err != nil {
		ctx.JSON(http.StatusNotFound, err)
		return
	}

	ctx.JSON(http.StatusOK, proxy)
}

func getProxyDelayFromProvider(ctx *zoox.Context) {
	url := ctx.Query().Get("url").String()
	timeout := ctx.Query().Get("timeout").Int()

	proxy, err := getProxyInProviderFromContext(ctx)
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

func getProxyInProviderFromContext(ctx *zoox.Context) (C.Proxy, error) {
	provider, err := getProviderFromContext(ctx)
	if err != nil {
		return nil, err
	}

	proxyName := ctx.Param().Get("name").String()

	proxy, ok := lo.Find(provider.Proxies(), func(proxy C.Proxy) bool {
		return proxy.Name() == proxyName
	})
	if !ok {
		return nil, ErrNotFound
	}

	return proxy, nil
}

func getProviderFromContext(ctx *zoox.Context) (provider.ProxyProvider, error) {
	name := ctx.Param().Get("providerName").String()
	providers := tunnel.Providers()
	provider, exist := providers[name]
	if !exist {
		return nil, ErrNotFound
	}

	return provider, nil
}

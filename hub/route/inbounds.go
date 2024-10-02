package route

import (
	"net/http"

	C "github.com/doreamon-design/clash/constant"
	"github.com/doreamon-design/clash/listener"
	"github.com/doreamon-design/clash/tunnel"
	"github.com/go-zoox/zoox"
)

func getInbounds(ctx *zoox.Context) {
	inbounds := listener.GetInbounds()

	ctx.JSON(http.StatusOK, zoox.H{
		"inbounds": inbounds,
	})
}

func updateInbounds(ctx *zoox.Context) {
	var req []C.Inbound
	if err := ctx.BindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrBadRequest)
		return
	}
	tcpIn := tunnel.TCPIn()
	udpIn := tunnel.UDPIn()
	listener.ReCreateListeners(req, tcpIn, udpIn)
	ctx.Status(http.StatusNoContent)
}

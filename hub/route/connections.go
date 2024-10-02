package route

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/doreamon-design/clash/tunnel/statistic"
	"github.com/go-zoox/zoox"

	"github.com/Dreamacro/protobytes"
	"github.com/gorilla/websocket"
)

func getConnections(ctx *zoox.Context) {
	if !websocket.IsWebSocketUpgrade(ctx.Request) {
		snapshot := statistic.DefaultManager.Snapshot()
		ctx.JSON(http.StatusOK, snapshot)
		return
	}

	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return
	}

	interval := ctx.Query().Get("interval").Int()
	if interval == 0 {
		interval = 1000
	}

	buf := protobytes.BytesWriter{}
	sendSnapshot := func() error {
		buf.Reset()
		snapshot := statistic.DefaultManager.Snapshot()
		if err := json.NewEncoder(&buf).Encode(snapshot); err != nil {
			return err
		}

		return conn.WriteMessage(websocket.TextMessage, buf.Bytes())
	}

	if err := sendSnapshot(); err != nil {
		return
	}

	tick := time.NewTicker(time.Millisecond * time.Duration(interval))
	defer tick.Stop()
	for range tick.C {
		if err := sendSnapshot(); err != nil {
			break
		}
	}
}

func closeConnection(ctx *zoox.Context) {
	id := ctx.Param().Get("id").String()
	snapshot := statistic.DefaultManager.Snapshot()
	for _, c := range snapshot.Connections {
		if id == c.ID() {
			c.Close()
			break
		}
	}

	ctx.Status(http.StatusNoContent)
}

func closeAllConnections(ctx *zoox.Context) {
	snapshot := statistic.DefaultManager.Snapshot()
	for _, c := range snapshot.Connections {
		c.Close()
	}

	ctx.Status(http.StatusNoContent)
}

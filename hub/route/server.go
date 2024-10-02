package route

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"unsafe"

	"github.com/doreamon-design/clash"
	C "github.com/doreamon-design/clash/constant"
	"github.com/doreamon-design/clash/log"
	"github.com/doreamon-design/clash/tunnel/statistic"
	"github.com/go-zoox/logger"
	"github.com/go-zoox/zoox"
	"github.com/go-zoox/zoox/defaults"
	"github.com/go-zoox/zoox/middleware"

	"github.com/Dreamacro/protobytes"
	"github.com/gorilla/websocket"
)

var (
	serverSecret = ""
	serverAddr   = ""

	uiPath = ""

	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

type Traffic struct {
	Up   int64 `json:"up"`
	Down int64 `json:"down"`
}

func SetUIPath(path string) {
	uiPath = C.Path.Resolve(path)
}

func Start(addr string, secret string) {
	if serverAddr != "" {
		return
	}

	serverAddr = addr
	serverSecret = secret

	app := defaults.Default()

	app.SetBanner(fmt.Sprintf(`
   ___                                      ___          _             _______         __ 
  / _ \___  _______ ___ ___ _  ___  ___    / _ \___ ___ (_)__ ____    / ___/ /__ ____ / / 
 / // / _ \/ __/ -_) _ '/  ' \/ _ \/ _ \  / // / -_|_-</ / _ '/ _ \  / /__/ / _ '(_-</ _ \
/____/\___/_/  \__/\_,_/_/_/_/\___/_//_/ /____/\__/___/_/\_, /_//_/  \___/_/\_,_/___/_//_/
                                                        /___/                              v%s																						                             
----------------------------------------------------------------
Maintainner: Zero (tobewhatwewant@gmail.com)
GitHub: https://github.com/doreamon-design/clash
`, clash.Version))

	// cors := cors.New(cors.Options{
	// 	AllowedOrigins: []string{"*"},
	// 	AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
	// 	AllowedHeaders: []string{"Content-Type", "Authorization"},
	// 	MaxAge:         300,
	// })
	// r.Use(cors.Handler)
	app.Use(middleware.CORS(&middleware.CorsConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowHeaders: []string{"Content-Type", "Authorization"},
		MaxAge:       300,
	}))

	app.Group("/", func(r *zoox.RouterGroup) {
		r.Use(authentication)

		r.Get("/", hello)
		r.Get("/logs", getLogs)
		r.Get("/traffic", traffic)
		r.Get("/version", version)
		r.Group("/configs", func(r *zoox.RouterGroup) {
			r.Get("/", getConfigs)
			r.Put("/", updateConfigs)
			r.Patch("/", patchConfigs)
		})
		r.Group("/inbounds", func(r *zoox.RouterGroup) {
			r.Get("/", getInbounds)
			r.Put("/", updateInbounds)
		})
		r.Group("/proxies", func(r *zoox.RouterGroup) {
			r.Get("/", getProxies)

			r.Group("/{name}", func(r *zoox.RouterGroup) {
				r.Get("/", getProxy)
				r.Get("/delay", getProxyDelay)
				r.Put("/", updateProxy)
			})
		})
		r.Group("/rules", func(r *zoox.RouterGroup) {
			r.Get("/", getRules)
		})
		r.Group("/connections", func(r *zoox.RouterGroup) {
			r.Get("/", getConnections)
			r.Delete("/", closeAllConnections)
			r.Delete("/{id}", closeConnection)
		})
		r.Group("/providers/proxies", func(r *zoox.RouterGroup) {
			r.Get("/", getProviders)

			r.Group("/{providerName}", func(r *zoox.RouterGroup) {
				r.Get("/", getProvider)
				r.Put("/", updateProvider)
				r.Get("/healthcheck", healthCheckProvider)

				r.Group("/{name}", func(r *zoox.RouterGroup) {
					r.Get("/", getProxyFromProvider)
					r.Get("/healthcheck", getProxyDelayFromProvider)
				})
			})
		})
		r.Group("/dns", func(r *zoox.RouterGroup) {
			r.Get("/query", queryDNS)
		})
	})

	if uiPath != "" {
		// r.Group(func(r chi.Router) {
		// 	fs := http.StripPrefix("/ui", http.FileServer(http.Dir(uiPath)))
		// 	r.Get("/ui", http.RedirectHandler("/ui/", http.StatusTemporaryRedirect).ServeHTTP)
		// 	r.Get("/ui/*", func(w http.ResponseWriter, r *http.Request) {
		// 		fs.ServeHTTP(w, r)
		// 	})
		// })

		// fs := http.StripPrefix("/ui", http.FileServer(http.Dir(uiPath)))
		// r.Get("/ui", zoox.WrapH(http.RedirectHandler("/ui/", http.StatusTemporaryRedirect)))
		// r.Get("/ui/*", zoox.WrapF(func(w http.ResponseWriter, r *http.Request) {
		// 	fs.ServeHTTP(w, r)
		// }))

		app.Static("/ui", uiPath)
	}

	if err := app.Run(addr); err != nil {
		logger.Errorf("External controller serve error: %s", err)
	}
}

func safeEuqal(a, b string) bool {
	aBuf := unsafe.Slice(unsafe.StringData(a), len(a))
	bBuf := unsafe.Slice(unsafe.StringData(b), len(b))
	return subtle.ConstantTimeCompare(aBuf, bBuf) == 1
}

func authentication(ctx *zoox.Context) {
	if serverSecret == "" {
		ctx.Next()
		return
	}

	// Browser websocket not support custom header
	if websocket.IsWebSocketUpgrade(ctx.Request) && ctx.Query().Get("token").String() != "" {
		token := ctx.Query().Get("token").String()
		if !safeEuqal(token, serverSecret) {
			ctx.JSON(http.StatusUnauthorized, ErrUnauthorized)
			return
		}

		ctx.Next()
		return
	}

	if token, found := ctx.BearerToken(); !found {
		ctx.JSON(http.StatusUnauthorized, ErrUnauthorized)
		return
	} else if ok := safeEuqal(token, serverSecret); !ok {
		ctx.JSON(http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	ctx.Next()
}

func hello(ctx *zoox.Context) {
	ctx.JSON(http.StatusOK, zoox.H{
		"hello":      "clash",
		"version":    clash.Version,
		"running_at": ctx.App.Runtime().RunningAt(),
	})
}

func traffic(ctx *zoox.Context) {
	var wsConn *websocket.Conn
	if websocket.IsWebSocketUpgrade(ctx.Request) {
		var err error
		wsConn, err = upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
		if err != nil {
			return
		}
	}

	if wsConn == nil {
		ctx.Header().Set("Content-Type", "application/json")
		ctx.Status(http.StatusOK)
	}

	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	t := statistic.DefaultManager
	buf := protobytes.BytesWriter{}
	var err error
	for range tick.C {
		buf.Reset()
		up, down := t.Now()
		if err := json.NewEncoder(&buf).Encode(Traffic{
			Up:   up,
			Down: down,
		}); err != nil {
			break
		}

		if wsConn == nil {
			_, err = ctx.Writer.Write(buf.Bytes())
			ctx.Writer.(http.Flusher).Flush()
		} else {
			err = wsConn.WriteMessage(websocket.TextMessage, buf.Bytes())
		}

		if err != nil {
			break
		}
	}
}

type Log struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

func getLogs(ctx *zoox.Context) {
	levelText := ctx.Query().Get("level").String()
	if levelText == "" {
		levelText = "info"
	}

	level, ok := log.LogLevelMapping[levelText]
	if !ok {
		ctx.JSON(http.StatusBadRequest, ErrBadRequest)
		return
	}

	var wsConn *websocket.Conn
	if websocket.IsWebSocketUpgrade(ctx.Request) {
		var err error
		wsConn, err = upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
		if err != nil {
			return
		}
	}

	if wsConn == nil {
		ctx.Header().Set("Content-Type", "application/json")
		ctx.Status(http.StatusOK)
	}

	ch := make(chan log.Event, 1024)
	sub := log.Subscribe()
	defer log.UnSubscribe(sub)
	buf := &bytes.Buffer{}

	go func() {
		for elm := range sub {
			log := elm.(log.Event)
			select {
			case ch <- log:
			default:
			}
		}
		close(ch)
	}()

	for log := range ch {
		if log.LogLevel < level {
			continue
		}
		buf.Reset()

		if err := json.NewEncoder(buf).Encode(Log{
			Type:    log.Type(),
			Payload: log.Payload,
		}); err != nil {
			break
		}

		var err error
		if wsConn == nil {
			_, err = ctx.Writer.Write(buf.Bytes())
			ctx.Writer.(http.Flusher).Flush()
		} else {
			err = wsConn.WriteMessage(websocket.TextMessage, buf.Bytes())
		}

		if err != nil {
			break
		}
	}
}

func version(ctx *zoox.Context) {
	ctx.JSON(http.StatusOK, zoox.H{"version": C.Version})
}

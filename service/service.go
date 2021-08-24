package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/creack/pty"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/skanehira/rtty/public"
	"github.com/skanehira/rtty/utils"
	"golang.org/x/net/websocket"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
)

// run command
var command = utils.GetEnv("SHELL", "bash")

// wait time for server start
var waitTime = 500
var checkProcInterval = 5

type ProxyItem struct {
	Name string `json:"name"`
	Url  string `json:"url"`
}

var proxyList []ProxyItem = []ProxyItem{
	{Name: "Prometheus", Url: "/proxy/9090"},
	{Name: "PMM", Url: "/proxy/9001"},
	{Name: "PMM", Url: "/proxy/9015"},
	{Name: "UWM", Url: "/proxy/9003"},
	{Name: "UWM", Url: "/proxy/8098"},
	{Name: "SCM", Url: "/proxy/9009"},
	{Name: "SCM", Url: "/proxy/9017"},
	{Name: "ODM", Url: "/proxy/9011"},
	{Name: "RMK", Url: "/proxy/8888"},
	{Name: "AlertManager", Url: "/proxy/9093"},
}

type Event string

const (
	EventResize  Event = "resize"
	EventSnedkey Event = "sendKey"
	EventClose   Event = "close"
)

type Message struct {
	Event Event
	Data  interface{}
}

var ptmx *os.File
var execCmd *exec.Cmd

type wsConn struct {
	conn *websocket.Conn
	buf  []byte
}

// Checking and buffering `b`
// If `b` is invalid UTF-8, it would be buffered
// if buffer is valid UTF-8, it would write to connection
func (ws *wsConn) Write(b []byte) (i int, err error) {
	if !utf8.Valid(b) {
		buflen := len(ws.buf)
		blen := len(b)
		ws.buf = append(ws.buf, b...)[:buflen+blen]
		if utf8.Valid(ws.buf) {
			_, e := ws.conn.Write(ws.buf)
			ws.buf = ws.buf[:0]
			return blen, e
		}
		return blen, nil
	}

	if len(ws.buf) > 0 {
		n, err := ws.conn.Write(ws.buf)
		ws.buf = ws.buf[:0]
		if err != nil {
			return n, err
		}
	}
	n, e := ws.conn.Write(b)
	return n, e
}

func serveWs(c echo.Context) (err error) {
	websocket.Handler(func(conn *websocket.Conn) {
		defer conn.Close()

		if ptmx == nil {
			var msg Message
			if err := json.NewDecoder(conn).Decode(&msg); err != nil {
				log.Println("failed to decode message:", err)
				return
			}

			rows, cols, err := utils.WindowSize(msg.Data)
			if err != nil {
				_, _ = conn.Write([]byte(fmt.Sprintf("%s\r\n", err)))
				return
			}
			winsize := &pty.Winsize{
				Rows: rows,
				Cols: cols,
			}

			c := utils.Filter(strings.Split(command, " "))
			if len(c) > 1 {
				//nolint
				execCmd = exec.Command(c[0], c[1:]...)
			} else {
				//nolint
				execCmd = exec.Command(c[0])
			}

			ptmx, err = pty.StartWithSize(execCmd, winsize)
			if err != nil {
				log.Println("failed to create pty", err)
				return
			}
		}
		go func() {
			// write data to process
			var msg Message

			for {
				err = json.NewDecoder(conn).Decode(&msg)
				if err != nil {
					log.Println("fail to decode json ", err)
					return
				}
				switch msg.Event {
				case EventClose:
					log.Println("close websocket")
					conn.Close()
					return
				case EventResize:
					rows, cols, err := utils.WindowSize(msg.Data)
					if err != nil {
						log.Println(err)
						return
					}

					winsize := &pty.Winsize{
						Rows: rows,
						Cols: cols,
					}

					if err := pty.Setsize(ptmx, winsize); err != nil {
						log.Println("failed to set window size:", err)
						return
					}
					continue

				case EventSnedkey:
					data, ok := msg.Data.(string)
					if !ok {
						log.Println("invalid message data:", data)
						return
					}

					_, err = ptmx.WriteString(data)
					if err != nil {
						log.Println("failed to write data to ptmx:", err)
						return
					}
				}
			}
		}()

		wsConn := &wsConn{
			conn: conn,
		}
		_, err = io.Copy(wsConn, ptmx)
		if err != nil {
			log.Println("fail to io copy ", err)
		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}

func StartWebService(addr, port, font, fontSize string, openView bool) (err error) {
	indexJS := public.IndexJS
	indexJS = strings.Replace(indexJS, "{addr}", template.JSEscapeString(addr), 1)
	indexJS = strings.Replace(indexJS, "{port}", port, 1)
	indexJS = strings.Replace(indexJS, "{fontFamily}", template.JSEscapeString(font), 1)
	indexJS = strings.Replace(indexJS, "{fontSize}", template.JSEscapeString(fontSize), 1)

	app := echo.New()
	app.Use(middleware.Recover())
	app.Use(middleware.Logger())
	app.GET("/", func(c echo.Context) error {
		return c.HTML(http.StatusOK, public.IndexHTML)
	})

	app.GET("ws", serveWs)

	app.GET("/css/*", func(c echo.Context) error {
		http.FileServer(http.FS(public.CssFiles)).ServeHTTP(c.Response(), c.Request())
		return nil
	})
	app.GET("/js/*", func(c echo.Context) error {
		http.FileServer(http.FS(public.JsFiles)).ServeHTTP(c.Response(), c.Request())
		return nil
	})
	app.GET("/index.js", func(c echo.Context) error {
		c.Response().Header().Set("Content-Type", "application/javascript")
		return c.String(http.StatusOK, indexJS)
	})

	app.Any("/proxy/:port/*", func(c echo.Context) error {
		port := c.Param("port")
		req := c.Request()
		path := strings.TrimPrefix(req.URL.Path, fmt.Sprintf("/proxy/%s", port))
		u := &url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("127.0.0.1:%s", port),
		}
		req.URL.Path = path
		req.RequestURI = req.URL.RequestURI()
		proxy := httputil.NewSingleHostReverseProxy(u)
		proxy.ServeHTTP(c.Response(), req)
		return nil
	})

	app.GET("/proxy-list", func(c echo.Context) error {
		return c.JSON(http.StatusOK, proxyList)
	})

	go func() {
		log.Println("running command: " + command)
		host := fmt.Sprintf("%s:%s", addr, port)
		log.Printf("running http://%s \n", host)

		if serverErr := app.Start(host); serverErr != nil {
			log.Println(serverErr)
		}
	}()

	// check process state
	go func() {
		ticker := time.NewTicker(time.Duration(checkProcInterval) * time.Second)
		for range ticker.C {
			if execCmd != nil {
				state, err := execCmd.Process.Wait()
				if err != nil {
					return
				}

				if state.ExitCode() != -1 {
					ptmx.Close()
					ptmx = nil
					execCmd = nil
				}
			}
		}
	}()

	// wait for run server
	time.Sleep(time.Duration(waitTime) * time.Microsecond)

	if openView {
		utils.OpenBrowser(fmt.Sprintf("http://%s:%s", addr, port))
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	if ptmx != nil {
		_ = ptmx.Close()
	}
	if execCmd != nil {
		_ = execCmd.Process.Kill()
		_, _ = execCmd.Process.Wait()
	}
	err = app.Shutdown(context.Background())

	return
}

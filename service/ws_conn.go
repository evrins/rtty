package service

import (
	"encoding/json"
	"fmt"
	"github.com/creack/pty"
	"github.com/labstack/echo/v4"
	"github.com/skanehira/rtty/utils"
	"golang.org/x/net/websocket"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type WsConn struct {
	conn *websocket.Conn
	buf  []byte
}

// Checking and buffering `b`
// If `b` is invalid UTF-8, it would be buffered
// if buffer is valid UTF-8, it would write to connection
func (ws *WsConn) Write(b []byte) (i int, err error) {
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

type SocketLimit struct {
	Limit int
	Count int
	sync.Mutex
}

func New(limit int) *SocketLimit {
	return &SocketLimit{Limit: limit}
}

func (s *SocketLimit) Incr() {
	s.Lock()
	defer s.Unlock()
	s.Count++
}

func (s *SocketLimit) Desc() {
	s.Lock()
	defer s.Unlock()
	s.Count--
}

func (s *SocketLimit) Exceed() bool {
	s.Lock()
	defer s.Unlock()
	return s.Count >= s.Limit
}

const limit = 256

var s = New(limit)

func ServeWs(c echo.Context) (err error) {
	websocket.Handler(func(conn *websocket.Conn) {
		if s.Exceed() {
			_, err = conn.Write([]byte(fmt.Sprintf("connection exceed limit %d\n", limit)))
			if err != nil {
				log.Println("fail to write exceed message ", err)
			}
			return
		}
		s.Incr()
		defer s.Desc()
		defer conn.Close()
		var ptmx *os.File
		var execCmd *exec.Cmd

		defaultSize := &pty.Winsize{
			Rows: 24,
			Cols: 80,
		}
		c := utils.Filter(strings.Split(command, " "))
		if len(c) > 1 {
			//nolint
			execCmd = exec.Command(c[0], c[1:]...)
		} else {
			//nolint
			execCmd = exec.Command(c[0])
		}
		ptmx, err = pty.StartWithSize(execCmd, defaultSize)
		if err != nil {
			log.Println("failed to create pty", err)
			return
		}
		defer func() {
			if ptmx != nil {
				_ = ptmx.Close()
			}
			if execCmd != nil {
				_ = execCmd.Process.Kill()
				_, _ = execCmd.Process.Wait()
			}
		}()
		// check process state
		go func() {
			ticker := time.NewTicker(time.Duration(checkProcInterval) * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				if execCmd == nil {
					continue
				}
				state, err := execCmd.Process.Wait()
				if err != nil {
					return
				}

				if state.ExitCode() != -1 {
					ptmx.Close()
				}
			}
		}()

		go func() {
			// read data from websocket
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

				case EventSendKey:
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

		// copy from process to websocket
		wsConn := &WsConn{
			conn: conn,
		}
		_, err = io.Copy(wsConn, ptmx)
		if err != nil {
			log.Println("fail to io copy ", err)
		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}

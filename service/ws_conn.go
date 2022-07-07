package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/creack/pty"
	"github.com/labstack/echo/v4"
	"github.com/skanehira/rtty/entity"
	"github.com/skanehira/rtty/utils"
	"io"
	"log"
	"nhooyr.io/websocket"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type WsConn struct {
	conn *websocket.Conn
	buf  []byte
}

func (ws *WsConn) Write(b []byte) (n int, err error) {
	safeMessage := base64.StdEncoding.EncodeToString(b)
	n = len(b)
	err = ws.conn.Write(context.TODO(), websocket.MessageText, append([]byte{Output}, []byte(safeMessage)...))
	return
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

func ServeWs(ctx echo.Context) (err error) {
	conn, err := websocket.Accept(ctx.Response(), ctx.Request(), &websocket.AcceptOptions{
		Subprotocols:       Protocols,
		InsecureSkipVerify: true,
	})
	if s.Exceed() {
		err = conn.Write(context.TODO(), websocket.MessageText, []byte(fmt.Sprintf("connection exceed limit %d\n", limit)))
		if err != nil {
			log.Println("fail to write exceed message ", err)
		}
		return
	}
	s.Incr()
	defer s.Desc()
	defer conn.Close(websocket.StatusNormalClosure, "")
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
		var msg []byte
		var err error
		var ts entity.TerminalSize

		for {
			_, msg, err = conn.Read(context.TODO())
			if err != nil {
				log.Println("fail to read from connection ", err)
				break
			}
			msgId := msg[0]
			switch msgId {
			case ResizeTerminal:
				err = json.Unmarshal(msg[1:], &ts)
				if err != nil {
					log.Println("fail to unmarshal terminal size")
					continue
				}

				winsize := &pty.Winsize{
					Rows: ts.Rows,
					Cols: ts.Columns,
				}

				if err := pty.Setsize(ptmx, winsize); err != nil {
					log.Println("failed to set window size:", err)
					return
				}

			case Input:
				data := msg[1:]
				_, err = ptmx.Write(data)
				if err != nil {
					log.Println("failed to write data to ptmx:", err)
					return
				}
			}
		}
	}()

	wsConn := &WsConn{
		conn: conn,
		buf:  make([]byte, 0),
	}

	_, err = io.Copy(wsConn, ptmx)
	if err != nil {
		log.Println("fail to io copy ", err)
	}

	return nil
}

package service

import (
	"context"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/skanehira/rtty/public"
	"github.com/skanehira/rtty/utils"
	"html/template"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
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

	app.GET("ws", ServeWs)

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

	// wait for run server
	time.Sleep(time.Duration(waitTime) * time.Microsecond)

	if openView {
		utils.OpenBrowser(fmt.Sprintf("http://%s:%s", addr, port))
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	err = app.Shutdown(context.Background())

	return
}

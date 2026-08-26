package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"ovencheck/internal/core"
	"ovencheck/internal/web"
	"strconv"
	"strings"
	"time"
)

func configuredAddr(flagAddr string) string {
	if flagAddr != "" {
		return flagAddr
	}
	if p := os.Getenv("PORT"); p != "" {
		if _, err := strconv.Atoi(p); err == nil {
			return "127.0.0.1:" + p
		}
	}
	return "127.0.0.1:19081"
}
func validateAddr(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("地址格式错误: %w", err)
	}
	if host == "" || host == "0.0.0.0" || host == "::" || strings.HasPrefix(host, "[") && !strings.Contains(host, "127.0.0.1") {
		return fmt.Errorf("服务必须绑定回环地址")
	}
	return nil
}
func main() {
	addr := flag.String("addr", "", "监听地址")
	self := flag.Bool("selfcheck", false, "执行自检后退出")
	state := flag.String("state", ".ovencheck/state.json", "快照路径")
	flag.Parse()
	listen := configuredAddr(*addr)
	if err := validateAddr(listen); err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	store, err := core.NewStore(*state)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	srv := web.NewServer(store)
	if *self {
		if err := selfCheck(listen, srv.Handler()); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("自检通过")
		return
	}
	server := &http.Server{Addr: listen, Handler: srv.Handler()}
	fmt.Println("服务监听 " + listen)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Println(err)
		os.Exit(1)
	}
}
func selfCheck(addr string, h http.Handler) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	server := &http.Server{Handler: h}
	go server.Serve(ln)
	defer server.Shutdown(context.Background())
	base := "http://" + ln.Addr().String()
	client := http.Client{Timeout: 2 * time.Second}
	for _, path := range []string{"/", "/api/batches"} {
		r, e := client.Get(base + path)
		if e != nil {
			return e
		}
		r.Body.Close()
		if r.StatusCode != 200 {
			return fmt.Errorf("自检 %s 返回 %d", path, r.StatusCode)
		}
	}
	return nil
}

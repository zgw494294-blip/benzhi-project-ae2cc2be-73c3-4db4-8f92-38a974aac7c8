package main

import (
	"errors"
	"net"
	"strconv"
	"strings"
)

type RuntimeConfig struct {
	Address   string
	StatePath string
	SelfCheck bool
}

func ParseRuntimeConfig(address, state string, selfCheck bool) (RuntimeConfig, error) {
	if address == "" {
		address = "127.0.0.1:19081"
	}
	if state == "" {
		state = ".ovencheck/state.json"
	}
	if err := validateRuntimeAddress(address); err != nil {
		return RuntimeConfig{}, err
	}
	return RuntimeConfig{Address: address, StatePath: state, SelfCheck: selfCheck}, nil
}

func validateRuntimeAddress(address string) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		return errors.New("监听地址必须是回环地址")
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 1024 || number > 65535 {
		return errors.New("端口必须在 1024-65535 之间")
	}
	if strings.Contains(host, "/") {
		return errors.New("地址包含非法路径")
	}
	return nil
}

func IsLoopbackAddress(address string) bool {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

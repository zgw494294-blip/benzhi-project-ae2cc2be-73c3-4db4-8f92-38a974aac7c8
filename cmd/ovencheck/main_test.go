package main

import "testing"

func TestConfiguredAddr(t *testing.T) {
	if configuredAddr("") != "127.0.0.1:19081" {
		t.Fatal("default address")
	}
	if configuredAddr("127.0.0.1:19100") != "127.0.0.1:19100" {
		t.Fatal("flag address")
	}
}

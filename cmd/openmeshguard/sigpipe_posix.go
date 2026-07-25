//go:build aix || darwin || dragonfly || freebsd || illumos || linux || netbsd || openbsd || solaris

package main

import (
	"os/signal"
	"syscall"
)

func ignoreBrokenPipeSignal() {
	signal.Ignore(syscall.SIGPIPE)
}

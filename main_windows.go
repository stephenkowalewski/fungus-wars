//go:build windows

package main

import (
	"crypto/tls"
	"log"
	"sync/atomic"
)

func certReloadHandler(certFile, keyFile string, currentCert *atomic.Pointer[tls.Certificate], logger *log.Logger) {
	// no-op on Windows
}

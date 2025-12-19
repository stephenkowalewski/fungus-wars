//go:build unix

package main

import (
	"crypto/tls"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
)

// reload the cert on SIGUSR1
func certReloadHandler(certFile, keyFile string, currentCert *atomic.Pointer[tls.Certificate], logger *log.Logger) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGUSR1)

	go func() {
		for range sigCh {
			logger.Println("Received SIGUSR1: reloading TLS certificate...")
			cert, err := tls.LoadX509KeyPair(certFile, keyFile)
			if err != nil {
				logger.Println("Failed to load x509 key pair:", err)
			} else {
				currentCert.Store(&cert)
				logger.Printf("Reloaded certificate with CN %s, valid until %s.",
					cert.Leaf.Subject.CommonName, cert.Leaf.NotAfter)
			}
		}
	}()
}

//go:build unix

package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestServerCertificateReloadOnSIGUSR1(t *testing.T) {
	// Generate a self-signed certificate, and save to temp files
	generatedCertCN := "initial-cert"
	certPEM, keyPEM, err := generateSelfSignedCert(generatedCertCN)
	if err != nil {
		t.Fatalf("failed to generate test certificate: %v", err)
	}
	certFile, err := os.CreateTemp("", "testcert-*.pem")
	if err != nil {
		t.Fatalf("failed to write cert file: %v", err)
	}
	keyFile, err := os.CreateTemp("", "testkey-*.pem")
	if err != nil {
		t.Fatalf("failed to write key file: %v", err)
	}
	if _, err := certFile.Write(certPEM); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	if _, err := keyFile.Write(keyPEM); err != nil {
		t.Fatalf("write key: %v", err)
	}
	certFile.Close()
	keyFile.Close()
	defer os.Remove(certFile.Name())
	defer os.Remove(keyFile.Name())

	// Start the server with a random port
	cmd := exec.Command(testBinaryPath,
		"--listen=127.0.0.1:0",
		"--docroot=./static",
		"--cert="+certFile.Name(),
		"--key="+keyFile.Name(),
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to capture stdout: %v", err)
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	// Ensure we kill the process even if test fails
	defer func() {
		_ = cmd.Process.Kill()
		cmd.Wait()
	}()

	// Read startup lines until we learn the chosen port
	startupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	lineCh := make(chan string, 1)
	errCh := make(chan error, 1)

	reader := bufio.NewReader(stdout)
	var addr string
	go func() {
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				errCh <- err
				return
			}
			line = strings.TrimSuffix(line, "\n")
			t.Log("server output: ", line)
			lineCh <- line
		}
	}()

	for {
		select {
		case <-startupCtx.Done():
			t.Fatalf("timeout waiting for server startup log line")

		case err := <-errCh:
			t.Fatalf("server exited early or failed to start: %v", err)

		case line := <-lineCh:
			if strings.Contains(line, "Starting HTTPS server on") {
				fields := strings.Fields(line)
				addr = fields[len(fields)-1]
				goto GotPort
			}
		}
	}

GotPort:
	if addr == "" {
		t.Fatalf("could not determine server bind address from logs")
	}

	// Give the server a moment to finish binding
	time.Sleep(150 * time.Millisecond)

	// Use a custom http client so we can skip certificate verification.
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	// verify we get back the original cert
	resp, err := client.Get("https://" + addr + "/")
	if err != nil {
		t.Fatalf("server did not respond: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %v", resp.Status)
	}
	if resp.TLS == nil {
		t.Fatalf("no TLS connection state")
	}
	if len(resp.TLS.PeerCertificates) == 0 {
		t.Fatalf("server returned no certificates")
	}
	serverCert := resp.TLS.PeerCertificates[0]
	if serverCert.Subject.CommonName != generatedCertCN {
		t.Fatalf("expected CommonName=%q, got %q", generatedCertCN, serverCert.Subject.CommonName)
	}

	// generate new cert
	generatedCertCN = "updated-cert"
	certPEM, keyPEM, err = generateSelfSignedCert(generatedCertCN)
	if err != nil {
		t.Fatalf("failed to generate test certificate: %v", err)
	}
	certFile, err = os.OpenFile(certFile.Name(), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		t.Fatalf("failed to write cert file: %v", err)
	}
	keyFile, err = os.OpenFile(keyFile.Name(), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		t.Fatalf("failed to write key file: %v", err)
	}
	if _, err := certFile.Write(certPEM); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	if _, err := keyFile.Write(keyPEM); err != nil {
		t.Fatalf("write key: %v", err)
	}
	certFile.Close()
	keyFile.Close()

	// reload
	if err := cmd.Process.Signal(syscall.SIGUSR1); err != nil {
		t.Fatalf("failed to send SIGUSR1: %v", err)
	}
	// Read serverlog to determine that the reload succeeded
	reloadCtx, cancelReload := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelReload()

	for {
		select {
		case <-reloadCtx.Done():
			t.Fatalf("timeout waiting for certificate reload log line")

		case err := <-errCh:
			t.Fatalf("server exited early during reload: %v", err)

		case line := <-lineCh:
			t.Log("server output: ", strings.TrimSuffix(line, "\n"))
			if strings.Contains(line, "Reloaded certificate with CN ") {
				goto ReloadComplete
			}
		}
	}
ReloadComplete:

	// Verify we get back the updated cert
	resp, err = client.Get("https://" + addr + "/")
	if err != nil {
		t.Fatalf("server did not respond: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %v", resp.Status)
	}
	if resp.TLS == nil {
		t.Fatalf("no TLS connection state")
	}
	if len(resp.TLS.PeerCertificates) == 0 {
		t.Fatalf("server returned no certificates")
	}
	serverCert = resp.TLS.PeerCertificates[0]
	if serverCert.Subject.CommonName != generatedCertCN {
		t.Fatalf("expected CommonName=%q, got %q", generatedCertCN, serverCert.Subject.CommonName)
	}
}

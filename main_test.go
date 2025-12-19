package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

var testBinaryPath = "./test-server-bin"

func TestMain(m *testing.M) {
	// Build once before all tests
	if runtime.GOOS == "windows" {
		testBinaryPath += ".exe"
	}
	build := exec.Command("go", "build", "-o", testBinaryPath, ".")
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		panic("failed to build test binary: " + err.Error())
	}

	// Run all tests
	code := m.Run()

	// Cleanup
	_ = os.Remove(testBinaryPath)
	os.Exit(code)
}

func TestServerStartsHTTP(t *testing.T) {
	// Start the server with a random port
	cmd := exec.Command(testBinaryPath,
		"--listen=127.0.0.1:0",
		"--docroot=./static",
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
			if strings.Contains(line, "Starting HTTP server on") {
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

	// Make a real HTTP request to ensure the server responds
	resp, err := http.Get("http://" + addr + "/")
	if err != nil {
		t.Fatalf("server did not respond: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %v", resp.Status)
	}
}

func TestServerStartsHTTPS(t *testing.T) {
	// Generate a self-signed certificate, and save to temp files
	generatedCertCN := "localhost"
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
	resp, err := client.Get("https://" + addr + "/")
	if err != nil {
		t.Fatalf("server did not respond: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %v", resp.Status)
	}

	// verify that the certificate we got back matches the one we generated
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
}

func generateSelfSignedCert(cn string) ([]byte, []byte, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: cn,
		},
		NotBefore: time.Now().Add(-1 * time.Hour),
		NotAfter:  time.Now().Add(24 * time.Hour),

		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	return certPEM, keyPEM, nil
}

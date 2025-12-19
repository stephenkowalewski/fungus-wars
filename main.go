package main

//go:generate go run game.go lobby.go player.go gen_js_vars.go
//go:generate go run gen_html_from_markdown.go

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/stephenkowalewski/fungus-wars/internal/logging"
	"github.com/stephenkowalewski/fungus-wars/internal/server_flags"

	"github.com/google/uuid"
)

var docroot string
var accesslog = log.New(os.Stdout, "", log.LstdFlags)
var serverlog = log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile)

var globalHeaders = server_flags.Header{
	"Cache-Control": []string{"max-age=0, no-cache, must-revalidate, proxy-revalidate"},
}
var staticContentHeaders = server_flags.Header{
	"Cache-Control": []string{"max-age=300, public"},
}
var debug bool
var printVersion bool

// These get set at build time with -ldflags
var (
	Version   = "dev"
	BuildDate = "unknown"
)

// handleGlobalheaders adds headers and calls another http.Handler
func handleGlobalheaders(staticContent bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// set global Headers
		for k, v := range globalHeaders {
			w.Header().Set(k, strings.Join(v, " "))
		}
		// set static Headers
		if staticContent {
			for k, v := range staticContentHeaders {
				w.Header().Set(k, strings.Join(v, " "))
			}
		}
		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

func getCookieWrapper(r *http.Request, name string) (string, error) {
	var val string
	c, err := r.Cookie(name)
	if err == nil {
		val = strings.ReplaceAll(c.Value, "+", " ")
	}
	val, err = url.QueryUnescape(val)
	return val, err
}

func clearCookies(w http.ResponseWriter) {
	w.Header().Add("Set-Cookie", `player-id=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/`)
	w.Header().Add("Set-Cookie", `player-name=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/`)
	w.Header().Add("Set-Cookie", `lobby-name=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/`)
	w.Header().Add("Set-Cookie", `game-id=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/`)
}

func main() {
	var listen, redirectListen, redirectTarget, redirectExclude string
	var certFile, keyFile string
	var accesslogger = server_flags.Logfile{Logger: &accesslog, Name: "stdout"}
	var serverlogger = server_flags.Logfile{Logger: &serverlog, Name: "stderr"}

	flag.StringVar(&listen, "listen", ":8080", "[ip]:port to bind to for game-related http(s) requests")
	flag.StringVar(&listen, "addr", ":8080", "alias for --listen")
	flag.StringVar(&certFile, "cert", "", "Path to PEM encoded x509 certificate file. Setting this flag enables https. If --cert is used, then --key is required.")
	flag.StringVar(&keyFile, "key", "", "Path to PEM encoded key file for use with --cert")
	flag.StringVar(&redirectListen, "http-redirect-addr", "", "[ip]:port to bind to for http to https redirects. Disabled if empty.")
	flag.StringVar(&redirectTarget, "http-redirect-target", "https://[[HOST]][[PATH]]", "Where to redirect clients to. [[HOST]] and [[PATH]] are replaced with the request Host header (no port) and URL Path, respectively.")
	flag.StringVar(&redirectExclude, "http-redirect-exclude", `^/\.well-known/acme-challenge/`, "Don't redirect paths matching this regex.")
	flag.StringVar(&docroot, "docroot", "./static", "directory to serve static assets from")
	flag.Var(&accesslogger, "accesslog", "log file for http requests")
	flag.Var(&serverlogger, "serverlog", "log file for server messages")
	flag.Var(&globalHeaders, "header", "Custom HTTP response header. May be specified more than once.")
	flag.Var(&staticContentHeaders, "static-header", "Custom HTTP response header for static pages. May be specified more than once. --static-header takes precedence over --header on static pages.")
	flag.BoolVar(&debug, "debug", false, "Enable extra debug logging. Initialize a lobby and game for testing.")
	flag.BoolVar(&printVersion, "version", false, "Print version info and exit")
	flag.Parse()

	if printVersion {
		fmt.Printf("%s version %s (built %s)\n", os.Args[0], Version, BuildDate)
		os.Exit(0)
	}

	docroot = filepath.FromSlash(docroot)

	if certFile == "" && keyFile != "" {
		serverlog.Fatal("A key file was given without a certificate file. Make sure to set both --cert and --key in order to enable TLS.")
	}
	if keyFile == "" && certFile != "" {
		serverlog.Fatal("A certificate file was given without a key file. Make sure to set both --cert and --key in order to enable TLS.")
	}

	s := &http.Server{
		Addr:           listen,
		Handler:        nil,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// configure TLS
	if certFile != "" {
		// load certificate
		var currentCert atomic.Pointer[tls.Certificate]
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			serverlog.Fatal("Failed to load x509 key pair: " + err.Error())
		}
		currentCert.Store(&cert)
		serverlog.Printf("Loaded certificate with CN %s, valid until %s.", cert.Leaf.Subject.CommonName, cert.Leaf.NotAfter)

		// reload certificate on SIGUSR1
		certReloadHandler(certFile, keyFile, &currentCert, serverlog)

		// add the TLS config to the server
		tlsConfig := &tls.Config{
			GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
				return currentCert.Load(), nil
			},
		}
		s.TLSConfig = tlsConfig
	}

	// start page
	http.Handle("/{$}",
		logging.AccessLogHandler(accesslog, handleGlobalheaders(true,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, docroot+"/index.html")
			}))))

	// lobby
	http.Handle("/lobby", // lobby landing page
		logging.AccessLogHandler(accesslog, handleGlobalheaders(true,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, docroot+"/lobby.html")
			}))))
	http.Handle("/lobby/join", // join new or existing lobby
		logging.AccessLogHandler(accesslog, handleGlobalheaders(false,
			http.HandlerFunc(lobbyJoinHandler))))
	http.Handle("/lobby/leave", // leave lobby
		logging.AccessLogHandler(accesslog, handleGlobalheaders(false,
			http.HandlerFunc(lobbyLeaveHandler))))
	http.Handle("/lobby/list", // get list of active lobbies
		logging.AccessLogHandler(accesslog, handleGlobalheaders(false,
			http.HandlerFunc(lobbyListHandler))))
	http.Handle("/lobby/get", // get the current state of the lobby the user is in
		logging.AccessLogHandler(accesslog, handleGlobalheaders(false,
			http.HandlerFunc(lobbyInfoHandler))))

	// game
	http.Handle("/game", // main game page
		logging.AccessLogHandler(accesslog, handleGlobalheaders(true,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, docroot+"/game.html")
			}))))
	http.Handle("/game/ws", // WebSocket for game communication
		logging.AccessLogHandler(accesslog, handleGlobalheaders(false,
			http.HandlerFunc(gameWsHandler))))
	http.Handle("/game/create", // creates a game
		logging.AccessLogHandler(accesslog, handleGlobalheaders(false,
			http.HandlerFunc(createGameHandler))))
	http.Handle("/game/join", // joins a game
		logging.AccessLogHandler(accesslog, handleGlobalheaders(false,
			http.HandlerFunc(joinGameHandler))))
	http.Handle("/game/leave", // leave a game
		logging.AccessLogHandler(accesslog, handleGlobalheaders(false,
			http.HandlerFunc(leaveGameHandler))))

	// static assets
	http.Handle("/static/",
		logging.AccessLogHandler(accesslog, handleGlobalheaders(true,
			http.StripPrefix("/static", http.FileServer(http.Dir(docroot))))))
	http.Handle("/favicon.ico",
		logging.AccessLogHandler(accesslog, handleGlobalheaders(true,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, docroot+"/favicon.ico")
			}))))
	http.Handle("/.well-known/",
		logging.AccessLogHandler(accesslog, handleGlobalheaders(true,
			http.FileServer(http.Dir(docroot)))))

	// default page - 404 Not Found
	http.Handle("/",
		logging.AccessLogHandler(accesslog, handleGlobalheaders(true,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "404 page not found", http.StatusNotFound)
			}))))

	if debug {
		addDebugEndpoints()

		createGameWithStaticUUIDs(uuid.Must(uuid.Parse("00000000-0000-0000-0002-000000000000")), 2)
		createGameWithStaticUUIDs(uuid.Must(uuid.Parse("00000000-0000-0000-0003-000000000000")), 3)
		createGameWithStaticUUIDs(uuid.Must(uuid.Parse("00000000-0000-0000-0004-000000000000")), 4)
	}

	// If redirectListen is set, start a server for HTTP to HTTPS redirects.
	if redirectListen != "" {
		// compile redirect exclude regex
		var excludeRegex *regexp.Regexp
		var err error
		if redirectExclude != "" {
			excludeRegex, err = regexp.Compile(redirectExclude)
		} else {
			excludeRegex, err = regexp.Compile(`^$no-match`)
		}
		if err != nil {
			serverlog.Fatalf("Failed to compile regex `%s`: %v", redirectExclude, err)
		}

		// define redirect handler
		redirectHandler := logging.AccessLogHandler(accesslog,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				target := strings.ReplaceAll(redirectTarget, "[[HOST]]", strings.Split(r.Host, ":")[0])
				target = strings.ReplaceAll(target, "[[PATH]]", r.URL.RequestURI())
				http.Redirect(w, r, target, http.StatusTemporaryRedirect)
			}),
		)

		// setup HTTP server for redirects
		redirectSrv := &http.Server{
			Addr:           redirectListen,
			ReadTimeout:    5 * time.Second,
			WriteTimeout:   5 * time.Second,
			MaxHeaderBytes: 1 << 20,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if excludeRegex.MatchString(r.URL.Path) {
					http.DefaultServeMux.ServeHTTP(w, r)
					return
				}
				redirectHandler.ServeHTTP(w, r)
			}),
		}

		go func() {
			serverlog.Println("Starting HTTP redirect server on " + redirectListen)
			if err := redirectSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				serverlog.Fatal(err)
			}
		}()
	}

	go cleanUpLobbiesBackgroundTask(serverlog, debug)
	go cleanUpGamesBackgroundTask(serverlog, debug)

	ln, err := net.Listen("tcp", listen)
	if err != nil {
		serverlog.Fatal(err)
	}
	// turns listen strings like "localhost:0" into something like "127.0.0.1:42189"
	listenAddr := ln.Addr().String()

	if certFile == "" {
		serverlog.Println("Starting HTTP server on " + listenAddr)
		serverlog.Fatal(s.Serve(ln))
	} else {
		serverlog.Println("Starting HTTPS server on " + listenAddr)
		tlsListener := tls.NewListener(ln, s.TLSConfig)
		serverlog.Fatal(s.Serve(tlsListener))
	}

}

package logging

import (
	"log"
	"net/http"
)

// AccesslogHandler logs requests and calls another http.Handler
func AccessLogHandler(l *log.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l.Printf("\t%s\t%s\t%s\t%s\t%s\n",
			r.RemoteAddr,
			r.Method,
			r.Host,
			r.RequestURI,
			r.Header.Get("Cookie"),
		)
		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

// LogWebSocket is called from within a WebSocket handler
func LogWebSocket(l *log.Logger, r *http.Request, msg any) {
	l.Printf("\t%s\t%s\t%s\t%s\t%s\t%v\n",
		r.RemoteAddr,
		"WebSocket",
		r.Host,
		r.RequestURI,
		r.Header.Get("Cookie"),
		msg,
	)
}

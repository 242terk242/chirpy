package main

import (
	"log"
	"net/http"
)

type Server struct {
	Addr string
}

func main() {
	// Use the http.NewServeMux() function to create an empty servemux.
	mux := http.NewServeMux()

	// Use the http.RedirectHandler() function to create a handler which 307
	// redirects all requests it receives to http://example.org.
	rh := http.RedirectHandler("http://example.org", 307)

	// Next we use the mux.Handle() function to register this with our new
	// servemux, so it acts as the handler for all incoming requests with the URL
	// path /foo.
	mux.Handle("/foo", rh)

	log.Print("Listening...")

	// Then we create a new server and start listening for incoming requests
	// with the http.ListenAndServe() function, passing in our servemux for it to
	// match requests against as the second argument.
	http.ListenAndServe(":8080", mux)
}

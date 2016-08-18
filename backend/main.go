package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"syscall"
	"path"
	"flag"
)

type Config struct {
	Address       string
	ListenAddress string
	Endpoint      string
}

func DefaultConfig() *Config {
	return &Config{
		Address: "http://127.0.0.1:4646",
		ListenAddress: ":3000",
		Endpoint: "/",
	}
}

var (
	flagAddress = flag.String("address", "", "The address of the Nomad server. " +
		"Overrides the NOMAD_ADDR environment variable if set. " +
		"(default: \"http://127.0.0.1:4646\")")
	flagListenAddress = flag.String("web.listen-address", "",
		"The address on which to expose the web interface. (default: \":3000\")")
	flagEndpoint = flag.String("web.path", "",
		"Path under which to expose the web interface. (default: \"/\")")
)

func (c *Config) Parse() {
	flag.Parse()

	address, ok := syscall.Getenv("NOMAD_ADDR")
	if ok {
		c.Address = address
	}
	if *flagAddress != "" {
		c.Address = *flagAddress
	}
	if *flagListenAddress != "" {
		c.ListenAddress = *flagListenAddress
	}
	if *flagEndpoint != "" {
		c.Endpoint = *flagEndpoint
	}
}

func main() {
	cfg := DefaultConfig()
	cfg.Parse()

	router := mux.NewRouter()

	broadcast := make(chan *Action)

	nomad := NewNomad(cfg.Address, broadcast)
	go nomad.watchAllocs()
	go nomad.watchEvals()
	go nomad.watchNodes()
	go nomad.watchJobs()

	hub := NewHub(nomad, broadcast)
	go hub.Run()

	router.HandleFunc(path.Join(cfg.Endpoint, "ws"), hub.Handler)
	router.PathPrefix(cfg.Endpoint).Handler(http.FileServer(assetFS()))

	log.Println("Starting server...")
	log.Fatal(http.ListenAndServe(cfg.ListenAddress, router))
}
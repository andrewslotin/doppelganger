package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

var (
	addr = flag.String("addr", "", "Listen address.")
	port = flag.Int("port", 8081, "Listen port.")
)

func main() {
	flag.Parse()

	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags)

	*addr = fmt.Sprintf("%s:%d", *addr, *port)
	log.Printf("doppelganger is listening on %s", *addr)
	http.ListenAndServe(*addr, nil)
}

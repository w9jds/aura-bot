package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	esi "github.com/w9jds/go.esi"
	evepraisal "github.com/w9jds/go.evepraisal"
	zkb "github.com/w9jds/zkb"
)

var (
	aura            *discordgo.Session
	httpClient      *http.Client
	esiClient       *esi.Client
	zkbClient       *zkb.Client
	appraisalClient *evepraisal.Client

	storage *Storage
)

func getEnv(id string, required bool) string {
	value := strings.Trim(os.Getenv(id), " ")
	if value == "" {
		log.Fatalf("Environment Variable %s is required", id)
	}

	return value
}

type CustomTransport struct {
	tripper http.RoundTripper
}

func (transport CustomTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	userAgent := getEnv("USER_AGENT", true)
	req.Header.Add("User-Agent", userAgent)
	return transport.tripper.RoundTrip(req)
}

func main() {
	httpClient = &http.Client{
		Transport: &CustomTransport{tripper: http.DefaultTransport},
	}

	esiClient = esi.CreateClient(httpClient)
	zkbClient = zkb.CreateClient(httpClient)
	appraisalClient = evepraisal.CreateClient(httpClient)

	storage = CreateStorage()
	defer storage.database.Close()

	aura = CreateAura()
	defer aura.Close()

	fetch()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	shutdown(aura)
}

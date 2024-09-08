package pooldataposter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"sync/atomic"
	"time"

	"golang.org/x/net/publicsuffix"
)

const (
	defaultSendInterval = 10 * time.Second
	minSendInterval     = 1 * time.Second
	defaultEndpointURL  = "http://localhost:8080/"
)

type IPPool struct {
	Name  string
	Start net.IP
	End   net.IP
}

type IPPoolInformer interface {
	GetIPPools() ([]IPPool, error)
}

type Poster struct {
	healthy        atomic.Bool
	Client         *http.Client
	IPPoolInformer IPPoolInformer
	SendInterval   time.Duration
	EndpointURL    string
}

type UnexpectedResponseError struct {
	StatusCode int
	Message    string
}

func (e UnexpectedResponseError) Error() string {
	return fmt.Sprintf("unexpected response from server, code: %d with message: %q", e.StatusCode, e.Message)
}

func (p *Poster) IsHealthy() bool {
	return p.healthy.Load()
}

func (p *Poster) init() {
	if p.Client == nil {
		jar, _ := cookiejar.New(&cookiejar.Options{
			PublicSuffixList: publicsuffix.List,
		})

		p.Client = &http.Client{
			Timeout: 10 * time.Second,
			Jar:     jar,
		}
	}

	if p.SendInterval <= minSendInterval {
		p.SendInterval = defaultSendInterval
	}

	if p.EndpointURL == "" {
		p.EndpointURL = defaultEndpointURL
	}
}

func (p *Poster) Run(ctx context.Context) {
	p.init()

loop:
	for {
		select {
		case <-time.After(p.SendInterval):
			pools, err := p.IPPoolInformer.GetIPPools()
			if err != nil {
				log.Println(err)
				p.healthy.Store(false)
				break
			}
			log.Printf("%v\n", pools)
			p.healthy.Store(true)

			err = p.postPoolData(ctx, pools)
			if err != nil {
				log.Println(err)
				p.healthy.Store(false)
			}
		case <-ctx.Done():
			log.Println("Shutdown...")
			break loop
		}
	}
}

func (p *Poster) postPoolData(ctx context.Context, pools []IPPool) error {
	body := &bytes.Buffer{}

	err := json.NewEncoder(body).Encode(pools)
	if err != nil {
		return fmt.Errorf("failed to encode pool: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.EndpointURL, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "close")

	resp, err := p.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		message := &bytes.Buffer{}
		message.ReadFrom(resp.Body)

		return &UnexpectedResponseError{
			StatusCode: resp.StatusCode,
			Message:    message.String(),
		}
	}
	return nil

}

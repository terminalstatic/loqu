package lib

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type Node struct {
	Host       string
	DestURL    string
	HealthPath string
	Active     bool
	Server     *http.Server
	ReqCount   int64
}

var servMutex sync.Mutex

func (n *Node) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	servMutex.Lock()
	if !n.Active {
		http.Error(w, http.StatusText(500), 500)
		return
	}
	servMutex.Unlock()

	atomic.AddInt64(&n.ReqCount, 1)
	defer atomic.AddInt64(&n.ReqCount, -1)

	req, err := http.NewRequest(r.Method, n.DestURL+r.URL.Path, nil)
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}

	defaultTtransport := &http.Transport{Proxy: nil}
	client := http.Client{Transport: defaultTtransport}
	res, err := client.Do(req)
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}

	for k, v := range res.Header {
		w.Header().Set(k, v[0])
	}

	body, err := ioutil.ReadAll(res.Body)

	defer res.Body.Close()
	w.Write(body)

}

func (nh *Node) Serve() {
	servMutex.Lock()
	nh.Server = &http.Server{Addr: nh.Host, Handler: nh}
	servMutex.Unlock()
	if err := nh.Server.ListenAndServe(); err != nil {
		fmt.Printf("\n%s\n> ", err)
	}
}

func (nh *Node) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := nh.Server.Shutdown(ctx); err != nil {
		fmt.Println(err)
		nh.Server.Close()
	}
	nh.Server = nil
}

func (nh *Node) ShutdownAndServe(nn *Node) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := nh.Server.Shutdown(ctx); err != nil {
		fmt.Println(err)
		nh.Server.Close()
	}
	nh.Server = nil
	nn.Serve()
}

func (nh *Node) SwitchTo(nn *Node) {

	servMutex.Lock()
	nn.Server = nh.Server
	nn.Server.Handler = nn
	nn.Active = true
	nh.Active = false
	servMutex.Unlock()
}

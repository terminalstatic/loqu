package lib

import (
	"context"
	"io/ioutil"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type Node struct {
	sync.RWMutex
	Host       string
	DestURL    string
	HealthPath string
	Active     bool
	Server     *http.Server
	ReqCount   int64
	LastStatus int64
}

func (n *Node) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	atomic.AddInt64(&n.ReqCount, 1)
	defer atomic.AddInt64(&n.ReqCount, -1)

	req, err := http.NewRequest(r.Method, n.DestURL+r.URL.Path, nil)
	if err != nil {
		atomic.StoreInt64(&n.LastStatus, int64(500))
		http.Error(w, http.StatusText(500), 500)
		return
	}

	defaultTtransport := &http.Transport{Proxy: nil}
	client := http.Client{Transport: defaultTtransport}
	res, err := client.Do(req)

	if err != nil {
		atomic.StoreInt64(&n.LastStatus, int64(500))
		http.Error(w, http.StatusText(500), 500)
		return
	}

	for k, v := range res.Header {
		w.Header().Set(k, v[0])
	}

	atomic.StoreInt64(&n.LastStatus, int64(res.StatusCode))
	w.WriteHeader(res.StatusCode)

	body, err := ioutil.ReadAll(res.Body)

	defer res.Body.Close()
	w.Write(body)

}

func (nh *Node) Serve() {
	nh.Lock()
	nh.Server = &http.Server{Addr: nh.Host, Handler: nh}
	nh.Unlock()
	if err := nh.Server.ListenAndServe(); err != nil {
	}
}

func (nh *Node) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := nh.Server.Shutdown(ctx); err != nil {
		nh.Server.Close()
	}
	nh.Server = nil
}

func (nh *Node) ShutdownAndServe(nn *Node) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := nh.Server.Shutdown(ctx); err != nil {
		nh.Server.Close()

	}
	nn.Lock()
	nn.Active = true
	nn.Unlock()
	nh.Lock()
	nh.Active = false
	nh.Unlock()
	nn.Serve()
}

func (nh *Node) SwitchTo(nn *Node) {
	nn.Lock()
	nh.RLock()
	nn.Server = nh.Server
	nn.Server.Handler = nn
	nn.Active = true
	nn.Unlock()
	nh.RUnlock()
	nh.Lock()
	nh.Active = false
	nh.Unlock()
}

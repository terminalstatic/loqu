package lib

import (
	"errors"
	"net"
	"net/http"
	"time"
)

func ProbeHttp(url string) error {
	r, err := http.Get(url)

	if err != nil {
		return err
	}

	if r.StatusCode != 200 {
		return errors.New("Http probe failed")
	}
	return nil
}

func ProbeTcp(host string) error {

	timeout := time.Second
	conn, err := net.DialTimeout("tcp", host, timeout)
	if err != nil {
		return nil
	}
	if conn != nil {
		defer conn.Close()
		return errors.New("Tcp probe failed")

	}

	return nil
}

package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
)

func ProxyHTTPS(rw http.ResponseWriter, req *http.Request) {
	var proxyConn, clientConn net.Conn
	var err error
	defer func() {
		if proxyConn != nil {
			log.Println("ProxyHTTPS() close proxyConn")
			proxyConn.Close()
		}

		if clientConn != nil {
			log.Println("ProxyHTTPS() close clientConn")
			clientConn.Close()
		}
	}()

	hij, ok := rw.(http.Hijacker)
	if !ok {
		log.Println("ProxyHTTPS() error")
		return
	}

	clientConn, _, err = hij.Hijack()
	if err != nil {
		log.Println(err)
		return
	}

	proxyConn, err = net.Dial("tcp", req.URL.Host)
	if err != nil {
		log.Println(err)
		return
	}

	_, err = clientConn.Write([]byte("HTTP/1.0 200 OK\r\n\r\n"))
	if err != nil {
		log.Println(err)
		return
	}

	go func() {
		io.Copy(clientConn, proxyConn)
	}()

	io.Copy(proxyConn, clientConn)
}

func ProxyHTTP(wr http.ResponseWriter, req *http.Request) {
	var resp *http.Response
	var err error

	defer func() {

		if err := recover(); err != nil {
			fmt.Println(err)
		}

		if resp != nil && resp.Body != nil {
			log.Println("ProxyHTTP() close resp.Body")
			resp.Body.Close()
		}
	}()

	clientHand := newClient()

	req.RequestURI = ""

	DelHopHeaders(&req.Header)

	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		AppendHostToXForwardHeader(&req.Header, &clientIP)
	}

	resp, err = clientHand.Do(req)
	if err != nil {
		log.Println("ProxyHTTP() error:", err)
		return
	}

	DelHopHeaders(&resp.Header)
	h := wr.Header()
	CopyHeader(&h, &resp.Header)
	wr.WriteHeader(resp.StatusCode)
	io.Copy(wr, resp.Body)
}

func newClient() *http.Client {
	httpClient := &http.Client{}
	return httpClient
}

func newInsecureSSLClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{Transport: tr}
	return httpClient
}

func HttpDispatcher(rw http.ResponseWriter, req *http.Request) {
	if req.URL.Port() == HTTPSPORT {
		ProxyHTTPS(rw, req)
	} else {
		ProxyHTTP(rw, req)
	}
}
package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"io"
	"log"
	"net"
	"time"

	"golang.org/x/sync/errgroup"
)

var (
	flagListenAddr  = flag.String("listen", ":9090", "the address to listen on")
	flagBackendAddr = flag.String("backend", "localhost:7251", "the backend server")
	flagKeyFilename = flag.String("key", "./server.key", "the private key filename")
	flagCrtFilename = flag.String("cert", "./server.crt", "the cert filename")
)

func main() {
	// load the flags
	flag.Parse()

	// load the certificates
	cert, err := tls.LoadX509KeyPair(*flagCrtFilename, *flagKeyFilename)
	if err != nil {
		log.Fatalf("LoadKeys: %s", err.Error())
	}

	// init the tls configs
	config := tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	// the listener
	ln, err := tls.Listen("tcp", *flagListenAddr, &config)
	if err != nil {
		log.Fatalf("Listner: %s", err.Error())
	}

	// let's play
	for {
		conn, err := ln.Accept()
		if err != nil {
			break
		}
		go handleConn(conn)
	}
}

// handle the incoming connection
func handleConn(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(time.Minute))

	d := &net.Dialer{
		Timeout: time.Second * 1,
	}
	client, err := d.Dial("tcp", *flagBackendAddr)
	if err != nil {
		log.Println("DialError: ", err.Error())
		return
	}
	defer client.Close()

	eg := &errgroup.Group{}
	eg.Go(func() error {
		_, err := io.Copy(client, conn)
		if err != nil {
			return err
		}
		return errors.New("server closed")
	})
	eg.Go(func() error {
		_, err := io.Copy(conn, client)
		if err != nil {
			return err
		}
		return errors.New("server closed")
	})
	err = eg.Wait()
	if err != nil {
		log.Println("ConnectionError: ", err.Error())
		return
	}
}

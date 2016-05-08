package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"github.com/rappet/monitorproxy/socks"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"log"

	"net"
	"net/http"
	"net/url"
	"strings"
)

const (
	license = `monitorproxy  Copyright (C) 2016  Raphael Peters (rappet)
This program comes with ABSOLUTELY NO WARRANTY; for details type 'show w'.
This is free software, and you are welcome to redistribute it
under certain conditions; type 'show c' for details.
`
)

var (
	bindAddr        = flag.String("addr", "localhost:1080", "Bind address (default localhost:1080)")
	sslRewriteHosts = make(map[string]bool)
)

func ScanHttp(reader io.Reader) {
	buffReader := bufio.NewReader(reader)
	line, err := buffReader.ReadString('\n')
	fmt.Println(line)
	for {
		_, err = buffReader.Discard(4096)
		if err != nil {
			return
		}
	}
}

func ImageFlipper(body []byte, response *http.Response) []byte {
	contentType := response.Header.Get("Content-Type")
	if contentType == "image/png" || contentType == "image/jpeg" {
		img, filetype, err := image.Decode(bytes.NewReader(body))
		fmt.Println("typ", filetype)
		if err == nil {
			maxX := img.Bounds().Max.X
			maxY := img.Bounds().Max.Y

			nImg := image.NewRGBA(img.Bounds())
			for x := 0; x <= maxX; x++ {
				for y := 0; y < maxY; y++ {
					nImg.Set(x, y, img.At(maxX-x, maxY-y))
				}
			}
			buffer := new(bytes.Buffer)
			if filetype == "png" {
				png.Encode(buffer, nImg)
			} else {
				jpeg.Encode(buffer, nImg, &jpeg.Options{Quality: 80})
			}
			return buffer.Bytes()
		}
	}
	return body
}

func InterceptHTTP(client, server bufio.ReadWriter, serverAddr url.URL) {
	request, err := http.ReadRequest(client.Reader)
	if err != nil {
		log.Println(err)
		return
	}
	request.Header.Del("Accept-Encoding")
	newurl, _ := url.Parse(request.URL.String())
	request.RequestURI = ""
	//orgUrl := request.URL
	newurl.Scheme = "http"
	newurl.Host = request.Host
	request.URL = newurl

	httpClient := &http.Client{}
	response, err := httpClient.Do(request)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println(response.Status)
	//if response.ContentLength <= 0 {
	//	response.ContentLength = 0
	//}
	//body := make([]byte, response.ContentLength)
	fmt.Println("-start-")
	body, _ := ioutil.ReadAll(response.Body)
	fmt.Println("-fin-")
	//io.ReadFull(response.Body, body)

	url := request.URL.String()
	fmt.Println(url, request.Host)
	contentType := response.Header.Get("Content-Type")
	fmt.Println(contentType)

	body = ImageFlipper(body, response)
	if strings.Contains(contentType, "text/html") {
		fmt.Println("-- REPLACING --")
		body = bytes.Replace(body, []byte("https://"), []byte("http://"), -1)
		//body = bytes.Replace(body, []byte("</body>"), []byte("<script type=text/javascript>alert(\"Active!\");</script></body>"), -1)
		body = bytes.Replace(body, []byte("Cloud"), []byte("Shit"), -1)
		body = bytes.Replace(body, []byte("cloud"), []byte("shit"), -1)
	}
	response.ContentLength = int64(len(body))

	response.Write(client)
	client.Write(body)
	server.Write(body)
	client.Flush()
	fmt.Println("end")
	response.Body.Close()
}

func main() {
	flag.Parse()
	log.Println(license)

	log.Printf("Binding on %s\n", *bindAddr)
	ln, err := net.Listen("tcp", *bindAddr)
	if err != nil {
		log.Fatalln(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("END IN ACCEPT LOOP")
			log.Fatalln(err)
		}
		go func() {
			buffconn := *bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
			header, err := socks.ReadHeader(buffconn)
			if err != nil {
				fmt.Println(err)
				conn.Close()
				return
			}
			fmt.Printf("%d %s\n", header.Version, header.IPAndPortAsString())
			server, err := net.Dial("tcp", header.IPAndPortAsString())
			if err != nil {
				conn.Close()
				fmt.Println("err")
				return
			}
			buffserver := *bufio.NewReadWriter(bufio.NewReader(server), bufio.NewWriter(server))
			socks.WriteResponse(buffconn, socks.ResponseGranted)
			buffconn.Flush()
			fmt.Println("Copying")
			if header.Port == 80 {
				InterceptHTTP(buffconn, buffserver, url.URL{Host: header.IPAndPortAsString()})
				conn.Close()
				server.Close()
			} else {
				go io.Copy(buffconn, server)
				go io.Copy(server, buffconn)
			}
		}()
	}
}

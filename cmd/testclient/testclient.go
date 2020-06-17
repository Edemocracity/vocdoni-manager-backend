package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"reflect"
	"strings"
	"time"

	json "github.com/rogpeppe/rjson"
	"gitlab.com/vocdoni/go-dvote/crypto/ethereum"
	"gitlab.com/vocdoni/go-dvote/log"
	"gitlab.com/vocdoni/vocdoni-manager-backend/types"
	"nhooyr.io/websocket"
)

// APIConnection holds an API websocket connection
type APIConnection struct {
	Conn *websocket.Conn
}

// NewAPIConnection starts a connection with the given endpoint address. The
// connection is closed automatically when the test or benchmark finishes.
func NewAPIConnection(addr string) *APIConnection {
	r := &APIConnection{}
	var err error
	r.Conn, _, err = websocket.Dial(context.TODO(), addr, nil)
	if err != nil {
		log.Fatal(err)
	}
	return r
}

// Request makes a request to the previously connected endpoint
func (r *APIConnection) Request(req types.MetaRequest, signer *ethereum.SignKeys) *types.MetaResponse {
	method := req.Method
	var cmReq types.RequestMessage
	cmReq.MetaRequest = req
	cmReq.ID = fmt.Sprintf("%d", rand.Intn(1000))
	cmReq.Timestamp = int32(time.Now().Unix())
	if signer != nil {
		var err error
		cmReq.Signature, err = signer.SignJSON(cmReq.MetaRequest)
		if err != nil {
			log.Fatalf("%s: %v", method, err)
		}
	}
	rawReq, err := json.Marshal(cmReq)
	if err != nil {
		log.Fatalf("%s: %v", method, err)
	}
	log.Infof("sending: %s", rawReq)
	if err := r.Conn.Write(context.TODO(), websocket.MessageText, rawReq); err != nil {
		log.Fatalf("%s: %v", method, err)
	}
	_, message, err := r.Conn.Read(context.TODO())
	log.Infof("received: %s", message)
	if err != nil {
		log.Fatalf("%s: %v", method, err)
	}
	var cmRes types.ResponseMessage
	if err := json.Unmarshal(message, &cmRes); err != nil {
		log.Fatalf("%s: %v", method, err)
	}
	if cmRes.ID != cmReq.ID {
		log.Fatalf("%s: %v", method, "request ID doesn'tb match")
	}
	if cmRes.Signature == "" {
		log.Fatalf("%s: empty signature in response: %s", method, message)
	}
	return &cmRes.MetaResponse
}

func printNice(resp *types.MetaResponse) {
	v := reflect.ValueOf(*resp)
	typeOfS := v.Type()
	output := "\n"
	var val reflect.Value
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).Type().Name() == "bool" || v.Field(i).Type().Name() == "int64" || !v.Field(i).IsZero() {
			if v.Field(i).Kind() == reflect.Ptr {
				val = v.Field(i).Elem()
			} else {
				val = v.Field(i)
			}
			output += fmt.Sprintf("%v: %v\n", typeOfS.Field(i).Name, val)
		}
	}
	fmt.Print(output + "\n")
}

func processLine(input []byte) types.MetaRequest {
	var req types.MetaRequest
	err := json.Unmarshal(input, &req)
	if err != nil {
		panic(err)
	}
	return req
}

func main() {
	host := flag.String("host", "ws://127.0.0.1:8000/api/registry", "host to connect to")
	logLevel := flag.String("logLevel", "error", "log level <debug, info, warn, error>")
	privKey := flag.String("key", "", "private key for signature (leave blank for auto-generate)")
	flag.Parse()
	log.Init(*logLevel, "stdout")
	rand.Seed(time.Now().UnixNano())

	signer := new(ethereum.SignKeys)
	if *privKey != "" {
		if err := signer.AddHexKey(*privKey); err != nil {
			panic(err)
		}
	} else {
		signer.Generate()
	}
	log.Infof("connecting to %s", *host)
	c := NewAPIConnection(*host)
	defer c.Conn.Close(websocket.StatusNormalClosure, "")
	var req types.MetaRequest
	reader := bufio.NewReader(os.Stdin)
	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		if len(line) < 7 || strings.HasPrefix(string(line), "#") {
			continue
		}
		req = processLine(line)

		resp := c.Request(req, signer)
		if !resp.Ok {
			printNice(resp)
		} else {
			printNice(resp)
		}

	}
}

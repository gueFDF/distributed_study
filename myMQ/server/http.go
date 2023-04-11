package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"myMQ/logs"
	"myMQ/message"
	"myMQ/protocol"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type ReqParams struct {
	params url.Values
	body   []byte
}

func NewReqParms(req *http.Request) (*ReqParams, error) {
	reqParams, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	return &ReqParams{reqParams, data}, nil
}

func (r *ReqParams) Query(key string) (string, error) {
	keyData := r.params[key]
	if len(keyData) == 0 {
		return "", errors.New("key not in query params")
	}
	return keyData[0], nil
}

func pingHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Length", "2")
	io.WriteString(w, "OK")
}

func putHandler(w http.ResponseWriter, req *http.Request) {
	reqParams, err := NewReqParms(req)
	if err != nil {
		logs.Error("HTTP: error - %s", err.Error())
		return
	}

	topicName, err := reqParams.Query("topic")
	if err != nil {
		logs.Error("HTTP: error -%s", err.Error())
		return
	}
	conn := &FakeConn{}
	client := NewClient(conn, "HTTP")
	proto := &protocol.Protocol{}
	resp, err := proto.Execute(client, "PUB", topicName, string(reqParams.body))
	if err != nil {
		logs.Error("HTTP: error - %s", err.Error())
		return
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(resp)))
	w.Write(resp)

}

func statsHandler(w http.ResponseWriter, req *http.Request) {
	for topicName := range message.TopicMap {
		io.WriteString(w, fmt.Sprintf("%s\n", topicName))
	}
}

func HttpServer(ctx context.Context, address string, port string, endChan chan struct{}) {
	http.HandleFunc("/ping", pingHandler)
	http.HandleFunc("/put", putHandler)
	http.HandleFunc("/stats", statsHandler)

	fqAddress := address + ":" + port

	httpServer := http.Server{
		Addr: fqAddress,
	}

	go func() {
		logs.Info("listening for http requests on %s", fqAddress)
		err := http.ListenAndServe(fqAddress, nil)
		if err != nil {
			logs.Fatal("http.ListenAndServe:", err)
		}
	}()

	<-ctx.Done()
	logs.Info("HTTP server on %s is shutdowning...", fqAddress)
	timeoutCtx, fn := context.WithTimeout(context.Background(), 10*time.Second)
	defer fn()
	if err := httpServer.Shutdown(timeoutCtx); err != nil {
		logs.Info("HTTP server shutduwn error: %v", err)
	}
	close(endChan)
}

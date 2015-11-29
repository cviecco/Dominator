package srpc

import (
	"io"
	"log"
	"net"
	"net/http"
	"path"
	"reflect"
	"strings"
)

const (
	connectString = "200 Connected to Go SRPC"
	rpcPath       = "/_goSRPC_/"
)

type receiverType struct {
	receiver reflect.Value
	methods  map[string]*reflect.Value
}

var receivers map[string]receiverType = make(map[string]receiverType)

// Precompute the reflect type for net.Conn. Can't use net.Conn directly
// because Typeof takes an empty interface value. This is annoying.
var typeOfConn = reflect.TypeOf((*net.Conn)(nil)).Elem()

func init() {
	http.HandleFunc(rpcPath, httpHandler)
}

func registerName(name string, rcvr interface{}) error {
	var receiver receiverType
	receiver.receiver = reflect.ValueOf(rcvr)
	receiver.methods = make(map[string]*reflect.Value)
	rcvrType := reflect.TypeOf(rcvr)
	for index := 0; index < rcvrType.NumMethod(); index++ {
		method := rcvrType.Method(index)
		if method.PkgPath != "" { // Method must be exported.
			continue
		}
		methodType := method.Type
		// Method needs two ins: receiver, net.Conn.
		if methodType.NumIn() != 2 {
			continue
		}
		if methodType.In(1) != typeOfConn {
			continue
		}
		if methodType.NumOut() != 0 {
			continue
		}
		receiver.methods[method.Name] = &method.Func
	}
	receivers[name] = receiver
	return nil
}

func httpHandler(w http.ResponseWriter, req *http.Request) {
	receiver, method, status := findMethod(w, req)
	if status != http.StatusOK {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(status)
		return
	}
	conn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		log.Println("rpc hijacking ", req.RemoteAddr, ": ", err.Error())
		return
	}
	defer conn.Close()
	io.WriteString(conn, "HTTP/1.0 "+connectString+"\n\n")
	method.Call([]reflect.Value{receiver.receiver, reflect.ValueOf(conn)})
}

func findMethod(w http.ResponseWriter, req *http.Request) (
	*receiverType, *reflect.Value, int) {
	if req.Method != "CONNECT" {
		return nil, nil, http.StatusMethodNotAllowed
	}
	rpcName := path.Base(req.URL.Path)
	splitRpcName := strings.Split(rpcName, ".")
	if len(splitRpcName) != 2 {
		return nil, nil, http.StatusBadRequest
	}
	receiver, ok := receivers[splitRpcName[0]]
	if !ok {
		return nil, nil, http.StatusNotFound
	}
	method, ok := receiver.methods[splitRpcName[1]]
	if !ok {
		return nil, nil, http.StatusNotFound
	}
	return &receiver, method, http.StatusOK
}
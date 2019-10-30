package main

import (
	"encoding/json"
	"fmt"
	"github.com/mDNSService/utils/nettool"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/iotdevice/zeroconf"
	"github.com/satori/go.uuid"
)

type response struct {
	code   int
	result map[string]*zeroconf.Server
	msg    string
}

type serviceInfo struct {
	Instance string   `json:"instance"`
	Service  string   `json:"service"`
	Domain   string   `json:"domain"`
	Port     int      `json:"port"`
	HostName string   `json:"host_name"`
	Ip       string   `json:"ip"`
	Text     []string `json:"text"`
}

var (
	s       = `{"code":%d,"result":[],"msg":"%s"}`
	router  = mux.NewRouter().StrictSlash(true)
	servers = make(map[string]*zeroconf.Server)
	//	TODO 端口检测功能（决定是否注册）
)

func main() {
	name := "mdns服务注册工具"
	model := "com.iotserv.services.mdnsResponser"
	//	添加服务
	router.HandleFunc("/addOne", addOne)
	//	删除服务
	router.HandleFunc("/deleteOne", deleteOne)
	//	查询服务
	router.HandleFunc("/getAll", getAll)
	port, err := nettool.GetOneFreeTcpPort()
	if err != nil {
		log.Fatal(err.Error())
		return
	}
	serverHttp := http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}
	fmt.Printf("%s(%s)开启在：http://127.0.0.1:%d/\n", name, model, port)
	exist, err := nettool.CheckComponentExist(model)
	if exist {
		fmt.Printf("当前网络存在%s(%s)，所以不进行开启", name, model)
		return
	}
	info := nettool.MDNSServiceBaseInfo
	info["name"] = name
	info["model"] = model
	s, err := nettool.RegistermDNSService(info, port)
	if err != nil {
		log.Fatal(err.Error())
		s.Shutdown()
		return
	}
	log.Println(serverHttp.ListenAndServe())
}

func addOne(w http.ResponseWriter, r *http.Request) {
	var newEntry *serviceInfo
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, s, 1, err.Error())
		return
	}
	err = json.Unmarshal(body, newEntry)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, s, 1, err.Error())
		return
	}
	server, err := zeroconf.RegisterProxy(newEntry.Instance, newEntry.Service, newEntry.Domain,
		newEntry.Port, newEntry.HostName, []string{newEntry.Ip}, newEntry.Text, nil)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, s, 1, err.Error())
		return
	}
	servers[uuid.Must(uuid.NewV4()).String()] = server
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, s, 0, "add mdns service ok")
}

func deleteOne(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if _, ok := servers[vars["id"]]; ok {
		servers[vars["id"]].Shutdown()
		delete(servers, vars["id"])
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, s, 0, "delete mdns service ok")
}

func getAll(w http.ResponseWriter, r *http.Request) {
	rst, err := json.Marshal(response{
		code:   1,
		result: servers,
		msg:    "ok",
	})
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, s, 1, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, string(rst))
}

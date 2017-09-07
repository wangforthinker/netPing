package utils

import (
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"net"
	"encoding/json"
)

type context struct {

}

type HandleFunc func(c *context, w http.ResponseWriter, r *http.Request)

var routes = map[string]map[string]HandleFunc{
	"POST": {
		"/saveLog":                                               serverSaveLog,
	},
}

func NewRouter(c *context) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)

	addRoute := func(url, method string, handleFunc HandleFunc) {
		logrus.Infof("register HTTP route %s %s", method, url)

		wraps := func(w http.ResponseWriter, r *http.Request) {
			logrus.Infof("HTTP request received %s %s %v ", r.Method, r.URL.String(), r.Body)
			handleFunc(c, w, r)
		}

		router.Path("/v{version:[0-9]+}" + url).Methods(method).HandlerFunc(wraps)
		router.Path(url).Methods(method).HandlerFunc(wraps)
	}

	for method, mapping := range routes {
		for url, handleFunc := range mapping {
			addRoute(url, method, handleFunc)
		}
	}

	return router
}

func NewServerAndRun(host string)  {
	listener, err := net.Listen("tcp", host)
	if err != nil {
		logrus.Fatal(err)
	}

	con := &context{}
	httpd := http.Server{
		Handler: NewRouter(con),
	}

	if err := httpd.Serve(listener); err != nil {
		logrus.Fatal(err)
	}
}

func writeJsonResponse(w http.ResponseWriter, statusCode int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(v)
}

func serverSaveLog(c *context, w http.ResponseWriter, r *http.Request)  {
	logs := &postLog{}
	err := json.NewDecoder(r.Body).Decode(&logs)
	if(err != nil) {
		logrus.Errorf("request error:%s", err.Error())
		writeJsonResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if(logs.Logs != nil && len(logs.Logs) > 0) {
		for _, log := range logs.Logs {
			if(log.LogType == InfoLog) {
				logrus.Infof("from server %s, message is %s", log.SourceIp, log.Msg)
			}else if(log.LogType == ErrorLog){
				logrus.Errorf("from server %s, message is %s", log.SourceIp, log.Msg)
			}else{
				logrus.Infof("from server %s, message is %s", log.SourceIp, log.Msg)
			}
		}
	}else{
		logrus.Errorf("receve logs is nil")
	}

	w.WriteHeader(http.StatusOK)
}
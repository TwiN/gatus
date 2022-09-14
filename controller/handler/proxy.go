package handler

import (
	"bytes"
	"github.com/TwiN/gatus/v4/config"
	"github.com/TwiN/gatus/v4/config/remote"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

func ProxyRequestHandler(cfg *config.Config) http.HandlerFunc {
	endpointName, _ := regexp.Compile("\"name\":\"([a-zA-Z0-9_\\- ]*)\"")
	endpointKey, _ := regexp.Compile("\"key\":\"([a-zA-Z0-9_-]*)\"")
	return func(writer http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		remoteInstance := GetRemoteInstanceByKey(cfg, vars["instanceKey"])
		proxyUrl, _ := url.Parse(remoteInstance.URL)
		instancePath := strings.Replace(r.URL.Path, "@"+remoteInstance.Key, "", 1)

		r.Host = proxyUrl.Host
		r.URL.Path = instancePath
		r.Header.Del("Accept-Encoding")

		proxy := httputil.NewSingleHostReverseProxy(proxyUrl)
		proxy.ModifyResponse = func(resp *http.Response) error {
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			err = resp.Body.Close()
			if err != nil {
				return err
			}
			b = endpointKey.ReplaceAll(b, []byte("\"key\":\"$1@"+remoteInstance.Key+"\""))
			b = endpointName.ReplaceAll(b, []byte("\"name\":\""+remoteInstance.EndpointPrefix+"$1\""))
			body := ioutil.NopCloser(bytes.NewReader(b))
			resp.Body = body
			resp.ContentLength = int64(len(b))
			resp.Header.Set("Content-Length", strconv.Itoa(len(b)))
			return nil
		}
		proxy.ServeHTTP(writer, r)
	}
}

func GetRemoteInstanceByKey(cfg *config.Config, instanceKey string) remote.Instance {
	var instance remote.Instance
	for _, i := range cfg.Remote.Instances {
		if i.Key == instanceKey {
			instance = i
		}
	}
	return instance
}

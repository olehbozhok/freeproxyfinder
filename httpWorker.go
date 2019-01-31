package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/olehbozhok/freeproxyfinder/parsers"
)

type ProxyWorker struct {
	wg  sync.WaitGroup
	mut sync.Mutex

	activeProxies []parsers.ProxySocks5Conf
}

func (pW *ProxyWorker) UpdateProxies() {
	proxies, err := parsers.GetProxiesListSpysOne("*")
	if err != nil {
		log.Printf("error parsers.GetProxiesListSpysOne err:%v\n", err)
		return
	}

	var activeProxies []parsers.ProxySocks5Conf
	addActiveProxy := func(proxy parsers.ProxySocks5Conf) {
		pW.mut.Lock()
		activeProxies = append(activeProxies, proxy)
		pW.mut.Unlock()
	}
	log.Printf("Got proxies %d\n", len(proxies))

	log.Printf("Run proxies check\n")
	wg := sync.WaitGroup{}
	wg.Add(len(proxies))
	for _, proxy := range proxies {

		go func(pr parsers.ProxySocks5Conf) {
			defer wg.Done()
			latency, err := pr.CheckLatency()
			if err != nil {
				log.Printf("error adress:%s err:%v\n", pr.Address, err)
			}
			if latency < 15.0 && err == nil {
				pr.LastCheckLatency = time.Now()
				addActiveProxy(pr)
			}

		}(proxy)
	}
	wg.Wait()
	log.Printf("find %d active proxies\n", len(activeProxies))

	pW.mut.Lock()
	pW.activeProxies = activeProxies
	pW.mut.Unlock()
}

func (pW *ProxyWorker) HttpHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	country := r.URL.Query().Get("country")

	// copy slice
	pW.mut.Lock()
	activeProxies := pW.activeProxies
	pW.mut.Unlock()

	if country == "" {
		data, err := json.Marshal(activeProxies)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(data)
		return
	}
	var filteredCountry []parsers.ProxySocks5Conf
	for _, proxy := range activeProxies {
		if proxy.CountryIsoCode == country {
			filteredCountry = append(filteredCountry, proxy)
		}
	}
	data, err := json.Marshal(filteredCountry)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(data)
	return
}

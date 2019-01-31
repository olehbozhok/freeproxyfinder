package parsers

import (
	"context"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/proxy"
)

type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
	DialWithConn(ctx context.Context, c net.Conn, network, address string) (net.Addr, error)
	Dial(network, address string) (net.Conn, error)
}

// ProxySocks5Conf represent config to proxy
type ProxySocks5Conf struct {
	Address   string `json:"address"`
	ProxyType string `json:"proxyType"`

	Latency          float64   `json:"latency"`
	LastCheckLatency time.Time `json:"lastCheckLatency"`
	// OutIP            net.IP

	CountryIsoCode string `json:"countryIsoCode"`

	dialer Dialer
}

// // IsOutIPFinded check if country and latency is finded
// func (pc *ProxySocks5Conf) IsOutIPFinded() bool {
// 	return len(pc.OutIP) != 0
// }

// IsContry check if country and latency is finded
func (pc *ProxySocks5Conf) IsContry() bool {
	return pc.CountryIsoCode != ""
}

func (pc *ProxySocks5Conf) GetDialer() (Dialer, error) {
	if pc.dialer != nil {
		return pc.dialer, nil
	}
	dialerSocksProxy, err := proxy.SOCKS5("tcp", pc.Address, nil, proxy.Direct)
	if err != nil {
		return nil, err
	}
	pc.dialer = dialerSocksProxy.(Dialer)
	return pc.dialer, nil
}

func (pc *ProxySocks5Conf) CheckLatency() (float64, error) {
	then := time.Now()
	dialer, err := pc.GetDialer()
	if err != nil {
		return 0, err
	}

	var netTransport = &http.Transport{
		DialContext:         dialer.DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	var netClient = &http.Client{
		Timeout:   time.Second * 15,
		Transport: netTransport,
	}
	// resp, err := netClient.Get("https://canihazip.com/s")
	// resp, err := netClient.Get("https://www.google.com/")

	// resp, err := http.DefaultClient.Get("https://canihazip.com/s")
	// resp, err := netClient.Get("https://canihazip.com/s")
	resp, err := netClient.Get("https://www.bbc.com/")

	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	outIPBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	_ = outIPBytes
	// pc.OutIP = outIPBytes
	pc.Latency = time.Now().Sub(then).Seconds()
	return pc.Latency, nil
}

package parsers

import (
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/robertkrimen/otto"
)

var (
	rePortChar = regexp.MustCompile(`((\d+|\w+)\^(\d+|\w+))`)
	reIP       = regexp.MustCompile(`\d+.\d+.\d+.\d+\:\d+`)
)

var client http.Client

func init() {
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatalf("Got error while creating cookie jar %s", err.Error())
	}

	client = http.Client{
		Jar:     jar,
		Timeout: time.Second * 30,
	}
}

// GetProxiesListSpysOne get proxy list from http://spys.one/en/socks-proxy-list/
func GetProxiesListSpysOne(countryISO string) (proxylist []ProxySocks5Conf, err error) {
	return getProxiesListSpysOneInternal("", countryISO)
}

func getProxiesListSpysOneInternal(xx0, countryISO string) (proxylist []ProxySocks5Conf, err error) {

	var req *http.Request

	URL := ""
	if countryISO == "" || countryISO == "*" {
		URL = "https://spys.one/en/socks-proxy-list/"
	} else {
		URL = fmt.Sprintf("https://spys.one/free-proxy-list/%s/", countryISO)
	}

	var header http.Header = map[string][]string{
		"User-Agent":      {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.5112.81 Safari/537.36"},
		"Accept":          {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8"},
		"Accept-Language": {"ru-RU,ru;q=0.8,en-US;q=0.5,en;q=0.3"},
		"Host":            {"spys.one"},
		"Referer":         {"http://spys.one/en/"},
		"origin":          {"https://spys.one"},
	}

	if xx0 == "" {
		req, err = http.NewRequest("GET", URL, nil)
	} else {
		form := url.Values{}
		form.Add("xx0", xx0)
		// 500 on page
		form.Add("xpp", "5")
		form.Add("xf1", "0")
		form.Add("xf2", "0")
		form.Add("xf4", "0")
		// socks5
		form.Add("xf5", "2")
		formData := form.Encode()
		req, err = http.NewRequest("POST", URL, strings.NewReader(formData))

		header.Add("Content-Type", "application/x-www-form-urlencoded")
		header.Add("Content-Length", strconv.Itoa(len(formData)))

	}
	if err != nil {
		return nil, err
	}

	req.Header = header

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	if xx0 == "" {
		xx0_1, ok := doc.Find(`body > table:nth-child(3) > tbody > tr:nth-child(4) > td > table > tbody > tr:nth-child(1) > td:nth-child(3) > input[type=hidden]`).Attr("value")
		if ok && xx0_1 != "" {
			return getProxiesListSpysOneInternal(xx0_1, countryISO)
		}
	}

	scriptText := doc.Find(`html body script`).Nodes[2].FirstChild.Data

	vm := otto.New()
	_, err = vm.Eval(scriptText)
	if err != nil {
		return nil, err
	}

	tableElemParser := func(i int, s *goquery.Selection) {
		var proxyConf ProxySocks5Conf

		s.Find("td").Each(func(i int, s *goquery.Selection) {
			switch i {
			case 0:
				// parse IP
				s = s.Find("td font.spy14")
				portStr := ""

				script := s.Find("script")

				// parse port
				portSubmatch := rePortChar.FindAllStringSubmatch(script.Text(), -1)
				for _, p := range portSubmatch {
					var valueStr string
					var value otto.Value
					if value, err = vm.Eval(p[1]); err != nil {
						fmt.Println(fmt.Errorf("could not get %s from js vm: %#v", p[1], err))
						continue
					}
					if valueStr, err = value.ToString(); err != nil {
						fmt.Println(fmt.Errorf("could not value.ToString() %s from js vm: %#v", p[1], err))
						continue
					}
					portStr += valueStr
				}
				script.Remove()

				proxyConf.Address = s.Text() + ":" + portStr
			case 1:
				proxyConf.ProxyType = s.Text()
			case 3:
				a := s.Find(`a`)
				href, exist := a.Attr("href")
				if exist {
					proxyConf.CountryIsoCode = strings.Replace((strings.Replace(href, "/free-proxy-list/", "", -1)), "/", "", -1)
				} else {
					s.Find(`font.spy1`).Remove()
					proxyConf.CountryIsoCode = s.Text()
				}
			}

		})
		if !reIP.MatchString(proxyConf.Address) {
			return
		}
		proxylist = append(proxylist, proxyConf)
	}

	doc.Find("html body table tbody tr td table tbody tr.spy1x").Each(tableElemParser)
	doc.Find("html body table tbody tr td table tbody tr.spy1xx").Each(tableElemParser)

	return proxylist, nil
}

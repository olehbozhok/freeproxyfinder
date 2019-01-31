package parsers

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/robertkrimen/otto"
)

var (
	rePortChar = regexp.MustCompile(`((\d+|\w+)\^(\d+|\w+))`)
	reIP       = regexp.MustCompile(`\d+.\d+.\d+.\d+\:\d+`)
)

// GetProxiesListSpysOne get proxy list from http://spys.one/en/socks-proxy-list/
func GetProxiesListSpysOne(countryISO string) (proxylist []ProxySocks5Conf, err error) {
	form := url.Values{}
	form.Add("xf1", "0")
	form.Add("xf2", "0")
	form.Add("xf4", "0")
	// socks5
	form.Add("xf5", "2")
	// 500 on page
	form.Add("xpp", "5")
	formData := form.Encode()

	var req *http.Request
	if countryISO == "" || countryISO == "*" {
		req, err = http.NewRequest("POST", "http://spys.one/en/socks-proxy-list/", strings.NewReader(formData))
	} else {
		req, err = http.NewRequest("POST", fmt.Sprintf("http://spys.one/free-proxy-list/%s/", countryISO), strings.NewReader(formData))
	}
	if err != nil {
		return nil, err
	}

	req.Header = map[string][]string{
		"User-Agent":      {"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:61.0) Gecko/20100101 Firefox/61.0"},
		"Accept":          {"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"},
		"Accept-Language": {"ru-RU,ru;q=0.8,en-US;q=0.5,en;q=0.3"},
		"Host":            {"spys.one"},
		"Referer":         {"http://spys.one/en/"},
		"Content-Type":    {"application/x-www-form-urlencoded"},
	}
	req.Header.Add("Content-Length", strconv.Itoa(len(formData)))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
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

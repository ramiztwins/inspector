package probers

import (
	"context"
	"fmt"
	"inspector/metrics"
	"inspector/mylogger"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

/*
 * This is an implementation of a prober called: basic http prober. It currently supports limited features, but should be
 * simple to extend from here.
 * Basic http prober currently exports these 3 metrics: connect_time, status and request_time.
 * TODO: add POST support, parameters support, HTTPS support to the basic http prober.
 */

type HTTPProber struct {
	TargetID       string
	ProberID       string
	Interval       time.Duration
	Url            string
	Method         string
	Parameters     map[string]string
	Cookies        map[string]string
	AllowRedirects bool
	Timeout        int 
	client         *http.Client
}

func (httpProber *HTTPProber) Initialize(targetID, proberID string) error {
	httpProber.TargetID = targetID
	httpProber.ProberID = proberID
	return nil
}

// Connect starts a new connection. We need a new connection on each Connect() invocation because we want to measure
// the connection time from scratch.
func (httpProber *HTTPProber) Connect(c chan metrics.SingleMetric) error {
	//TODO: handle https urls in httpProber
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			start := time.Now()
			conn, err := net.Dial(network, addr)
			if err != nil {
				mylogger.MainLogger.Errorf("Connection Failed for URL %s. Method: %s. Error: %s",
					httpProber.Url, httpProber.Method, err)
				return nil, err
			}
			c <- metrics.CreateSingleMetric("connect_time", time.Since(start).Milliseconds(), nil,
				map[string]string{
					"target_id": httpProber.getTargetID(),
					"prober_id": httpProber.getProberID(),
				})
			return conn, nil
		},
		DisableKeepAlives: true,
	}
	httpProber.client = &http.Client{
		Timeout:   time.Duration(httpProber.Timeout) * time.Second,
		Transport: transport,
	}

	// The default http client follows redirects 10 levels deep.
	// Client should not follow http redirects if instructed by the config
	if !httpProber.AllowRedirects {
		httpProber.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	// Initialize cookies
	baseURL, _ := url.Parse(httpProber.Url)
	jar, err := cookiejar.New(nil)
	if err != nil {
		mylogger.MainLogger.Error("Could not initialize cookie jar in http prober")
		return err
	}
	var cookies []*http.Cookie
	for key, value := range httpProber.Cookies {
		cookies = append(cookies, &http.Cookie{
			Name:  key,
			Value: value})
	}
	jar.SetCookies(baseURL, cookies)
	httpProber.client.Jar = jar
	return nil
}

func (httpProber *HTTPProber) RunOnce(c chan metrics.SingleMetric) error {
	var response *http.Response
	var err error
	var start time.Time

	params := url.Values{}
	for name, value := range httpProber.Parameters {
		params.Add(name, value)
	}
	baseURL, _ := url.Parse(httpProber.Url)
	baseURL.RawQuery = params.Encode()

	if httpProber.Method == "GET" {
		start = time.Now()
		response, err = httpProber.client.Get(baseURL.String())
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unsupported method: %s", httpProber.Method)
	}

	c <- metrics.CreateSingleMetric("response_time", time.Since(start).Milliseconds(), nil,
		map[string]string{
			"target_id": httpProber.getTargetID(),
			"prober_id": httpProber.getProberID(),
		})

	c <- metrics.CreateSingleMetric("status", int64(response.StatusCode), nil,
		map[string]string{
			"target_id": httpProber.getTargetID(),
			"prober_id": httpProber.getProberID(),
		})

	if response.TLS != nil {
		c <- metrics.CreateSingleMetric("certificate_expiration",
			int64(response.TLS.PeerCertificates[0].NotAfter.Sub(time.Now()).Hours())/24, nil,
			map[string]string{
				"target_id": httpProber.getTargetID(),
				"prober_id": httpProber.getProberID(),
			})
	}

	response.Body.Close()
	return nil
}

func (httpProber *HTTPProber) TearDown() error {
	httpProber.client.CloseIdleConnections()
	return nil
}

func (httpProber *HTTPProber) getTargetID() string {
	return httpProber.TargetID
}

func (httpProber *HTTPProber) getProberID() string {
	return httpProber.ProberID
}

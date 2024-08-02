package probers

import (
	"crypto/tls"
	"inspector/metrics"
	"inspector/mylogger"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptrace"
	"net/url"
	"time"
)

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
	var dnsStart, connectStart time.Time
	var dnsDuration, connectDuration time.Duration

	// Setup HTTP transport and trace to capture DNS and Connect timings
	transport := &http.Transport{
		DisableKeepAlives: true,
	}

	trace := &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			dnsStart = time.Now()
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			dnsDuration = time.Since(dnsStart)
		},
		ConnectStart: func(network, addr string) {
			connectStart = time.Now()
		},
		ConnectDone: func(network, addr string, err error) {
			connectDuration = time.Since(connectStart)
		},
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

	// Create a request with tracing enabled
	req, _ := http.NewRequest(httpProber.Method, httpProber.Url, nil)
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	// Perform the request to measure the connect time and DNS lookup
	resp, err := httpProber.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Report DNS lookup time
	c <- metrics.CreateSingleMetric("dns_lookup_time", dnsDuration.Milliseconds(), nil,
		map[string]string{
			"target_id": httpProber.getTargetID(),
			"prober_id": httpProber.getProberID(),
		})

	// Report Connect time
	c <- metrics.CreateSingleMetric("connect_time", connectDuration.Milliseconds(), nil,
		map[string]string{
			"target_id": httpProber.getTargetID(),
			"prober_id": httpProber.getProberID(),
		})

	return nil
}

func (httpProber *HTTPProber) RunOnce(c chan metrics.SingleMetric) error {
	var response *http.Response
	var err error
	var start, tlsStart, firstByteStart, lastByteTime time.Time
	var tlsDuration, firstByteDuration, totalDownloadDuration time.Duration
	var contentSize int64

	params := url.Values{}
	for name, value := range httpProber.Parameters {
		params.Add(name, value)
	}
	baseURL, _ := url.Parse(httpProber.Url)
	baseURL.RawQuery = params.Encode()

	// Set up httptrace to measure TLS handshake and TTFB timings
	trace := &httptrace.ClientTrace{
		TLSHandshakeStart: func() {
			tlsStart = time.Now()
		},
		TLSHandshakeDone: func(cs tls.ConnectionState, err error) {
			tlsDuration = time.Since(tlsStart)
		},
		GotFirstResponseByte: func() {
			firstByteDuration = time.Since(firstByteStart)
		},
	}

	req, _ := http.NewRequest(httpProber.Method, baseURL.String(), nil)
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	start = time.Now()
	firstByteStart = start

	response, err = httpProber.client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	// Measure the content size and time to last byte
	contentSize, err = io.Copy(io.Discard, response.Body)
	if err != nil {
		return err
	}
	lastByteTime = time.Now()
	totalDownloadDuration = lastByteTime.Sub(firstByteStart)

	// Report TLS handshake time
	c <- metrics.CreateSingleMetric("tls_handshake_time", tlsDuration.Milliseconds(), nil,
		map[string]string{
			"target_id": httpProber.getTargetID(),
			"prober_id": httpProber.getProberID(),
		})

	// Report Time to First Byte
	c <- metrics.CreateSingleMetric("time_to_first_byte", firstByteDuration.Milliseconds(), nil,
		map[string]string{
			"target_id": httpProber.getTargetID(),
			"prober_id": httpProber.getProberID(),
		})

	// Report full response time
	c <- metrics.CreateSingleMetric("response_time", time.Since(start).Milliseconds(), nil,
		map[string]string{
			"target_id": httpProber.getTargetID(),
			"prober_id": httpProber.getProberID(),
		})

	// Report HTTP status code
	c <- metrics.CreateSingleMetric("status", int64(response.StatusCode), nil,
		map[string]string{
			"target_id": httpProber.getTargetID(),
			"prober_id": httpProber.getProberID(),
		})

	// Report Time to Last Byte
	c <- metrics.CreateSingleMetric("time_to_last_byte", totalDownloadDuration.Milliseconds(), nil,
		map[string]string{
			"target_id": httpProber.getTargetID(),
			"prober_id": httpProber.getProberID(),
		})

	/* Report Content Size
	* TODO: Handle situation when content_size equals 0.
	* Redirects, status code with no body, empty responses, content filtering/blocking
	*/
	c <- metrics.CreateSingleMetric("content_size", contentSize, nil,
		map[string]string{
			"target_id": httpProber.getTargetID(),
			"prober_id": httpProber.getProberID(),
		})

	// Report certificate expiration time if HTTPS
	if response.TLS != nil {
		c <- metrics.CreateSingleMetric("certificate_expiration",
			int64(response.TLS.PeerCertificates[0].NotAfter.Sub(time.Now()).Hours())/24, nil,
			map[string]string{
				"target_id": httpProber.getTargetID(),
				"prober_id": httpProber.getProberID(),
			})
	}

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

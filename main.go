package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"
)

func main() {
	addr := flag.String("addr", ":8080", "address to listen on")
	flag.Parse()
	fs := http.FileServer(
		http.FS(CaseInsensitiveFS{resources}),
	)
	handler := func(w http.ResponseWriter, r *http.Request) {
		if path.Clean(r.URL.Path) == "/cors" {
			proxyRequest(w, r)
			return
		}
		fs.ServeHTTP(w, r)
	}
	log.Println("listening on", *addr)
	http.ListenAndServe(*addr, http.HandlerFunc(handler))
}

var cache sync.Map // string -> cacheEntry

type cacheEntry struct {
	Expires time.Time
	Content []byte
	sync.RWMutex
}

func proxyRequest(w http.ResponseWriter, r *http.Request) {
	url, err := url.Parse(r.URL.Query().Get("u"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	switch url.Host {
	case "forecast.weather.gov",
		"api.weather.com",
		"www.aviationweather.gov",
		"www.wunderground.com",
		"api-ak.wunderground.com",
		"tidesandcurrents.noaa.gov",
		"l-36.com",
		"airquality.weather.gov",
		"airnow.gov",
		"www.airnowapi.org",
		"alerts.weather.gov",
		"mesonet.agron.iastate.edu",
		"tgftp.nws.noaa.gov",
		"www.cpc.ncep.noaa.gov",
		"radar.weather.gov",
		"www2.ehs.niu.edu",
		"api.usno.navy.mil":
	default:
		http.Error(w, "invalid host", http.StatusBadRequest)
		return
	}
	content, err := getURL(*url)
	if err != nil {
		log.Println("error serving CORS request:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(content)
	return
}

func getURL(u url.URL) ([]byte, error) {
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, errors.New("unsupported scheme")
	}
	v, ok := cache.LoadOrStore(u.String(), &cacheEntry{})
	cacheEntry := v.(*cacheEntry)
	if ok {
		cacheEntry.RLock()
		exp := cacheEntry.Expires
		content := cacheEntry.Content
		cacheEntry.RUnlock()
		if time.Now().Before(exp) {
			return content, nil
		}
	}
	cacheEntry.Lock()
	defer cacheEntry.Unlock()
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Origin", u.Scheme+"://"+u.Host)
	if u.Host == "api.weather.gov" {
		req.Header.Set("User-Agent", "(WeatherStar 4000+/v1 (https://battaglia.ddns.net/twc; vbguyny@gmail.com)")
		req.Header.Set("Accept", "application/vnd.noaa.dwml+xml")
	} else {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36")
	}
	const maxRetries = 3
	var body io.ReadCloser
	for retries := 0; retries < maxRetries; retries++ {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			continue
		}
		if resp.StatusCode != http.StatusOK {
			if resp.StatusCode >= 500 && resp.StatusCode < 600 {
				continue
			}
			return nil, fmt.Errorf("%s says: %s", u.Host, resp.Status)
		}
		body = resp.Body
		break
	}
	if body == nil {
		return nil, errors.New("failed to get")
	}
	defer body.Close()
	content, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	expiryDuration := time.Hour
	switch u.Host {
	case "www2.ehs.niu.edu":
		expiryDuration = 5 * time.Minute
	case "api.usno.navy.mil":
		expiryDuration = 3 * time.Hour
	}
	cacheEntry.Content = content
	cacheEntry.Expires = time.Now().Add(expiryDuration)
	return content, nil
}

type CaseInsensitiveFS struct {
	fs fs.FS
}

func (cifs CaseInsensitiveFS) Open(name string) (fs.File, error) {
	lower := strings.ToLower(name)
	var file fs.File
	found := errors.New("file found")
	fn := func(path string, d fs.DirEntry, err error) error {
		lowerp := strings.ToLower(path)
		if lowerp == lower {
			var err error
			file, err = cifs.fs.Open(path)
			if err != nil {
				return err
			}
			return found
		}
		if path != "." && d.IsDir() && !strings.HasPrefix(lower, lowerp) {
			return fs.SkipDir
		}
		return nil
	}
	err := fs.WalkDir(cifs.fs, ".", fn)
	if err != found {
		return nil, fs.ErrNotExist
	}
	return file, nil
}

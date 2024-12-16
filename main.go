package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/fatih/color"
)

var (
	SEARCH_ENGINES = map[string]struct {
		URL    string
		Params func(dork string, page int) map[string]string
	}{
		"google": {
			URL: "https://www.google.com/search",
			Params: func(dork string, page int) map[string]string {
				return map[string]string{
					"q":     dork,
					"start": fmt.Sprintf("%d", page*10),
				}
			},
		},
		"bing": {
			URL: "https://www.bing.com/search",
			Params: func(dork string, page int) map[string]string {
				return map[string]string{
					"q":     dork,
					"first": fmt.Sprintf("%d", page*10+1),
				}
			},
		},
		"duckduckgo": {
			URL: "https://duckduckgo.com/html/",
			Params: func(dork string, page int) map[string]string {
				return map[string]string{
					"q": dork,
					"s": fmt.Sprintf("%d", page*30),
				}
			},
		},
	}

	USER_AGENTS = []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/85.0.4183.102 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0 Safari/605.1.15",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.96 Safari/537.36",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0 Mobile/15A372 Safari/604.1",
		"Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:85.0) Gecko/20100101 Firefox/85.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:92.0) Gecko/20100101 Firefox/92.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 11_2_3) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.3 Safari/605.1.15",
		"Mozilla/5.0 (iPad; CPU OS 14_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0 Mobile/15E148 Safari/604.1",
	}

	logger      *log.Logger
	results     sync.Map
	validProxies []Proxy
)

type Proxy struct {
	URL      string
	Original string
	Client   *http.Client
}

type ProxyCheckResult struct {
	HTTP  bool
	HTTPS bool
}

func initLogger() {
	logger = log.New(os.Stdout, "", log.LstdFlags)
}

func getRandomUserAgent() string {
	return USER_AGENTS[rand.Intn(len(USER_AGENTS))]
}

func parseProxy(proxyStr string) *Proxy {
	pattern := regexp.MustCompile(`(?:(?P<username>[^:@]+):(?P<password>[^@]+)@)?(?P<ip>[^:]+):(?P<port>\d+)`)
	matches := pattern.FindStringSubmatch(proxyStr)
	if len(matches) == 0 {
		logger.Println(color.YellowString("Proxy format tidak valid: %s", proxyStr))
		return nil
	}
	proxyURL := ""
	if matches[1] != "" && matches[2] != "" {
		proxyURL = fmt.Sprintf("http://%s:%s@%s:%s", matches[1], matches[2], matches[3], matches[4])
	} else {
		proxyURL = fmt.Sprintf("http://%s:%s", matches[3], matches[4])
	}

	transport := &http.Transport{
		Proxy: func(_ *http.Request) (*url.URL, error) {
			return url.Parse(proxyURL)
		},
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       90 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	return &Proxy{
		URL:      proxyURL,
		Original: proxyStr,
		Client:   client,
	}
}

func loadProxies(filePath string) []Proxy {
	var proxies []Proxy
	absPath, _ := filepath.Abs(filePath)
	logger.Println(color.CyanString("Memeriksa keberadaan file proxy di: %s", absPath))
	file, err := os.Open(filePath)
	if err != nil {
		logger.Println(color.RedString("File proxy '%s' tidak ditemukan.", filePath))
		return proxies
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		proxyStr := strings.TrimSpace(scanner.Text())
		if proxyStr != "" && proxyStr != "0.0.0.0:80" {
			proxy := parseProxy(proxyStr)
			if proxy != nil {
				proxies = append(proxies, *proxy)
			}
		}
	}
	logger.Println(color.GreenString("%d proxy dimuat dari '%s'", len(proxies), filePath))
	return proxies
}

func extractDomain(rawURL string) string {
	// Hapus karakter setelah spasi atau simbol yang tidak valid
	cleanedURL := strings.Split(rawURL, " ")[0]
	cleanedURL = strings.Split(cleanedURL, "›")[0]
	parsed, err := url.Parse(cleanedURL)
	if err != nil {
		logger.Println(color.RedString("Gagal mengekstrak domain dari %s: %v", rawURL, err))
		return ""
	}
	host := parsed.Hostname() // Mengambil hostname tanpa port
	host = strings.TrimPrefix(host, "www.")
	if host == "" {
		logger.Println(color.YellowString("Hostname kosong setelah ekstraksi dari URL: %s", rawURL))
		return ""
	}
	// Tambahkan pengecualian untuk menghindari domain internal seperti bing.com
	if host == "bing.com" {
		return ""
	}
	logger.Println(color.CyanString("Ekstrak domain: %s dari URL: %s", host, rawURL))
	return host
}

func validateProxy(proxy *Proxy) ProxyCheckResult {
	result := ProxyCheckResult{}
	testHTTP := "http://httpbin.org/ip"
	testHTTPS := "https://httpbin.org/ip"

	// Uji HTTP
	req, _ := http.NewRequest("GET", testHTTP, nil)
	req.Header.Set("User-Agent", getRandomUserAgent())
	resp, err := proxy.Client.Do(req)
	if err == nil && resp.StatusCode == 200 {
		result.HTTP = true
		logger.Println(color.GreenString("[LIVE HTTP] %s - Status Code: %d", proxy.Original, resp.StatusCode))
	} else {
		logger.Println(color.YellowString("[DEAD HTTP] %s - Error: %v", proxy.Original, err))
	}

	// Uji HTTPS
	req, _ = http.NewRequest("GET", testHTTPS, nil)
	req.Header.Set("User-Agent", getRandomUserAgent())
	resp, err = proxy.Client.Do(req)
	if err == nil && resp.StatusCode == 200 {
		result.HTTPS = true
		logger.Println(color.GreenString("[LIVE HTTPS] %s - Status Code: %d", proxy.Original, resp.StatusCode))
	} else {
		logger.Println(color.YellowString("[DEAD HTTPS] %s - Error: %v", proxy.Original, err))
	}

	return result
}

func validateProxies(proxies []Proxy) []Proxy {
	var valid []Proxy
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 200) // Meningkatkan batas concurrent validations
	mu := &sync.Mutex{}

	for i := range proxies {
		wg.Add(1)
		go func(p *Proxy) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			result := validateProxy(p)
			if result.HTTPS {
				p.URL = fmt.Sprintf("https://%s", p.Original)
				mu.Lock()
				valid = append(valid, *p)
				mu.Unlock()
			} else if result.HTTP {
				p.URL = fmt.Sprintf("http://%s", p.Original)
				mu.Lock()
				valid = append(valid, *p)
				mu.Unlock()
			}
		}(&proxies[i])
	}
	wg.Wait()
	logger.Println(color.GreenString("%d proxy valid ditemukan setelah validasi.", len(valid)))
	return valid
}

func fetchURL(client *http.Client, url string, headers map[string]string) (string, int, error) {
	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return "", 0, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp.StatusCode, err
	}
	return string(body), resp.StatusCode, nil
}

func searchGoogle(client *http.Client, dork string, page int) []string {
	engine := "google"
	baseURL := SEARCH_ENGINES[engine].URL
	params := SEARCH_ENGINES[engine].Params(dork, page)
	u, _ := url.Parse(baseURL)
	q := u.Query()
	for key, value := range params {
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()
	logger.Println(color.GreenString("Mencari Google: Dork '%s', Halaman %d", dork, page+1))

	html, status, err := fetchURL(client, u.String(), map[string]string{
		"User-Agent": getRandomUserAgent(),
	})
	if err != nil || status != 200 {
		logger.Println(color.YellowString("Gagal mengambil Google untuk dork '%s', halaman %d", dork, page+1))
		if html != "" {
			logger.Println(color.MagentaString("HTML yang diterima:\n%s", html))
		}
		return nil
	}
	if strings.Contains(strings.ToLower(html), "detected unusual traffic") || strings.Contains(strings.ToLower(html), "captcha") {
		logger.Println(color.YellowString("CAPTCHA terdeteksi di Google untuk dork '%s', halaman %d. Mencoba dengan proxy lain.", dork, page+1))
		return nil
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		logger.Println(color.RedString("Gagal parsing HTML Google: %v", err))
		return nil
	}
	var domains []string
	doc.Find("div.yuRUbf > a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists && (strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://")) {
			domain := extractDomain(href)
			if domain != "" {
				domains = append(domains, domain)
			}
		}
	})
	if len(domains) == 0 {
		re := regexp.MustCompile(`https?://[^\s"<>]+`)
		matches := re.FindAllString(html, -1)
		for _, href := range matches {
			domain := extractDomain(href)
			if domain != "" {
				domains = append(domains, domain)
			}
		}
	}
	logger.Println(color.CyanString("Google: Dork '%s', Halaman %d menemukan %d domain", dork, page+1, len(domains)))
	return domains
}

func searchBing(client *http.Client, dork string, page int) []string {
	engine := "bing"
	baseURL := SEARCH_ENGINES[engine].URL
	params := SEARCH_ENGINES[engine].Params(dork, page)
	u, _ := url.Parse(baseURL)
	q := u.Query()
	for key, value := range params {
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()
	logger.Println(color.GreenString("Mencari Bing: Dork '%s', Halaman %d", dork, page+1))

	html, status, err := fetchURL(client, u.String(), map[string]string{
		"User-Agent": getRandomUserAgent(),
	})
	if err != nil || status != 200 {
		logger.Println(color.YellowString("Gagal mengambil Bing untuk dork '%s', halaman %d", dork, page+1))
		if html != "" {
			logger.Println(color.MagentaString("HTML yang diterima:\n%s", html))
		}
		return nil
	}
	if strings.Contains(strings.ToLower(html), "unusual traffic") || strings.Contains(strings.ToLower(html), "captcha") {
		logger.Println(color.YellowString("CAPTCHA terdeteksi di Bing untuk dork '%s', halaman %d. Mencoba dengan proxy lain.", dork, page+1))
		return nil
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		logger.Println(color.RedString("Gagal parsing HTML Bing: %v", err))
		return nil
	}
	var domains []string
	doc.Find("li.b_algo a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists && (strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://")) && !strings.Contains(href, "bing.com/ck/a") {
			logger.Println(color.BlueString("Link ditemukan: %s", href))
			domain := extractDomain(href)
			if domain != "" {
				domains = append(domains, domain)
			}
		}
	})
	if len(domains) == 0 {
		re := regexp.MustCompile(`https?://[^\s"<>]+`)
		matches := re.FindAllString(html, -1)
		for _, href := range matches {
			domain := extractDomain(href)
			if domain != "" {
				domains = append(domains, domain)
			}
		}
	}
	logger.Println(color.CyanString("Bing: Dork '%s', Halaman %d menemukan %d domain", dork, page+1, len(domains)))
	return domains
}

func searchDuckDuckGo(client *http.Client, dork string, page int) []string {
	engine := "duckduckgo"
	baseURL := SEARCH_ENGINES[engine].URL
	params := SEARCH_ENGINES[engine].Params(dork, page)
	u, _ := url.Parse(baseURL)
	q := u.Query()
	for key, value := range params {
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()
	logger.Println(color.GreenString("Mencari DuckDuckGo: Dork '%s', Halaman %d", dork, page+1))

	html, status, err := fetchURL(client, u.String(), map[string]string{
		"User-Agent": getRandomUserAgent(),
	})
	if err != nil || status != 200 {
		logger.Println(color.YellowString("Gagal mengambil DuckDuckGo untuk dork '%s', halaman %d", dork, page+1))
		if html != "" {
			logger.Println(color.MagentaString("HTML yang diterima:\n%s", html))
		}
		return nil
	}
	if strings.Contains(strings.ToLower(html), "robot check") || strings.Contains(strings.ToLower(html), "captcha") {
		logger.Println(color.YellowString("CAPTCHA terdeteksi di DuckDuckGo untuk dork '%s', halaman %d. Mencoba dengan proxy lain.", dork, page+1))
		return nil
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		logger.Println(color.RedString("Gagal parsing HTML DuckDuckGo: %v", err))
		return nil
	}
	var domains []string
	doc.Find("a.result__a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists && (strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://")) {
			domain := extractDomain(href)
			if domain != "" {
				domains = append(domains, domain)
			}
		}
	})
	if len(domains) == 0 {
		re := regexp.MustCompile(`https?://[^\s"<>]+`)
		matches := re.FindAllString(html, -1)
		for _, href := range matches {
			domain := extractDomain(href)
			if domain != "" {
				domains = append(domains, domain)
			}
		}
	}
	logger.Println(color.CyanString("DuckDuckGo: Dork '%s', Halaman %d menemukan %d domain", dork, page+1, len(domains)))
	return domains
}

func processSearch(engine, dork string, page int, proxy *Proxy, retries int) []string {
	for i := 0; i <= retries; i++ {
		var currentDomains []string
		switch engine {
		case "google":
			currentDomains = searchGoogle(proxy.Client, dork, page)
		case "bing":
			currentDomains = searchBing(proxy.Client, dork, page)
		case "duckduckgo":
			currentDomains = searchDuckDuckGo(proxy.Client, dork, page)
		default:
			logger.Println(color.RedString("Mesin pencari tidak didukung: %s", engine))
			return nil
		}
		if len(currentDomains) > 0 {
			for _, domain := range currentDomains {
				results.Store(domain, struct{}{})
			}
			return currentDomains
		}
		if i < retries {
			logger.Println(color.YellowString("Retrying dork '%s', mesin '%s', halaman %d dengan proxy lain.", dork, engine, page+1))
			// Pilih proxy baru untuk retry
			if len(validProxies) == 0 {
				logger.Println(color.RedString("Tidak ada proxy yang valid untuk retry. Keluar dari proses ini."))
				return nil
			}
			proxy = &validProxies[rand.Intn(len(validProxies))]
			// Tambahkan delay eksponensial
			delay := time.Duration(2<<i) * time.Second
			time.Sleep(delay)
		}
	}
	return nil
}

func loadDorks(filePath string) []string {
	var dorks []string
	file, err := os.Open(filePath)
	if err != nil {
		logger.Println(color.RedString("File dork '%s' tidak ditemukan.", filePath))
		return dorks
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		dork := strings.TrimSpace(scanner.Text())
		if dork != "" {
			dorks = append(dorks, dork)
		}
	}
	logger.Println(color.GreenString("Memulai pencarian untuk %d dork.", len(dorks)))
	return dorks
}

func saveResults(outputFile string) {
	file, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logger.Println(color.RedString("Gagal membuka file output '%s': %v", outputFile, err))
		return
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	results.Range(func(key, value interface{}) bool {
		fmt.Fprintln(writer, key.(string))
		return true
	})
	writer.Flush()
	logger.Println(color.GreenString("Hasil disimpan di %s", outputFile))
}

func main() {
	initLogger()
	rand.Seed(time.Now().UnixNano())

	dorkFile := flag.String("d", "", "Path ke file dork.txt untuk multiple dorks")
	pages := flag.Int("p", 1, "Tentukan Jumlah Halaman (Default: 1)")
	output := flag.String("o", "results.txt", "Tentukan File Output (Default: results.txt)")
	engines := flag.String("e", "google", "Tentukan Mesin Pencari yang digunakan (Default: google). Pisahkan dengan koma jika lebih dari satu (contoh: google,bing,duckduckgo)")
	threads := flag.Int("t", 500, "Jumlah permintaan simultan (Default: 500)")
	proxyFile := flag.String("x", "proxy.txt", "Path ke file proxy.txt (Default: proxy.txt)")
	retries := flag.Int("r", 3, "Jumlah maksimal retry per permintaan (Default: 3)")
	flag.Parse()

	if *dorkFile == "" {
		logger.Println(color.RedString("File dork wajib diisi. Gunakan -d untuk menentukan file dork."))
		return
	}

	// Load Dorks terlebih dahulu sebelum digunakan dalam logging
	dorks := loadDorks(*dorkFile)
	if len(dorks) == 0 {
		logger.Println(color.RedString("File dork '%s' kosong atau tidak valid.", *dorkFile))
		return
	}

	banner := `
 _____  _____  _____  __ ___ ___  _____  _____  _____ 
 |  _  \/  _  \/  _  \|  |  //___\/  _  \/   __\|__   /
 |  |  ||  |  ||  _  <|  _ < |   ||  |  ||  |_ | /  _/ 
 |_____/\_____/\__|\_/|__|__\\___/\__|__/\_____//_____|                                 
./SultanZio Version 1.2
`
	fmt.Println(banner)

	logger.Println(color.GreenString("Memulai pencarian untuk %d dork dengan %d halaman masing-masing menggunakan rotating proxies.", len(dorks), *pages))

	proxies := loadProxies(*proxyFile)
	if len(proxies) == 0 {
		logger.Println(color.RedString("Tidak ada proxy yang dimuat. Pastikan file proxy.txt memiliki daftar proxy yang valid."))
		return
	}

	validProxies = validateProxies(proxies)
	if len(validProxies) == 0 {
		logger.Println(color.RedString("Tidak ada proxy yang valid. Keluar dari program."))
		return
	}
	logger.Println(color.GreenString("%d proxy valid ditemukan.", len(validProxies)))

	var wg sync.WaitGroup
	engineList := strings.Split(*engines, ",")
	engineMap := []string{}
	for _, eng := range engineList {
		eng = strings.TrimSpace(strings.ToLower(eng))
		if _, exists := SEARCH_ENGINES[eng]; exists {
			engineMap = append(engineMap, eng)
		}
	}
	if len(engineMap) == 0 {
		engineMap = append(engineMap, "google")
	}
	pagesList := *pages
	semaphore := make(chan struct{}, *threads)

	for _, dork := range dorks {
		for _, engine := range engineMap {
			for page := 0; page < pagesList; page++ {
				wg.Add(1)
				go func(d string, e string, p int) {
					defer wg.Done()
					semaphore <- struct{}{}
					defer func() { <-semaphore }()
					// Pilih proxy secara acak
					proxy := &validProxies[rand.Intn(len(validProxies))]
					processSearch(e, d, p, proxy, *retries)
					// Tambahkan delay acak antara 1-2 detik untuk menghindari deteksi
					delay := time.Duration(rand.Intn(2)+1) * time.Second
					time.Sleep(delay)
				}(dork, engine, page)
			}
		}
	}
	wg.Wait()
	saveResults(*output)
	totalDomains := 0
	results.Range(func(key, value interface{}) bool {
		totalDomains++
		return true
	})
	logger.Println(color.GreenString("Total Dorks: %d", len(dorks)))
	logger.Println(color.GreenString("Total Halaman per Dork: %d", *pages))
	logger.Println(color.GreenString("Total Domain Unik: %d", totalDomains))
}

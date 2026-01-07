package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

const (
	HTML_CACHE         = "../cache"
	IMAGE_CACHE        = "../images"
	INPUT_CSV          = "scrape.csv"
	KOOPI_HOME_URL     = "https://www.kupi.cz"
	KOOPI_IMAGE_URL    = "https://img.kupi.cz"
	KOOPI_SEARCH_URL   = "https://www.kupi.cz/hledej?f="
	KOOPI_SUBPAGE      = "&page="
	LOCK_FILE          = "/tmp/koopi.lock"
	LOCK_FILE_DURATION = time.Hour
	MAX_SCRAPED_GOODS  = 500
	MAX_THREADS        = 3
	OUTPUT_CSV         = "koopi.csv"
	OUTPUT_JSON        = "koopi.json"
	REQ_TIMEOUT        = 10 * time.Second
)

// UA strings
var UserAgents = []string{
	"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Linux; Android 10; LM-Q720) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Linux; Android 11; CPH2251) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Linux; Android 12; SM-A525F) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Linux; Android 12; V2134) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Linux; Android 13; M2101K6G) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Linux; Android 13; SM-G991U) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Linux; Android 13; SM-S908E) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Linux; Android 14; Pixel 8 Pro) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.6045.159 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/78.0.3904.108 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.149 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.85 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/94.0.4606.81 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Ubuntu; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.6167.85 Safari/537.36",
	"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:120.0.1) Gecko/20100101 Firefox/120.0.1",
	"Mozilla/5.0 (iPad; CPU OS 16_7_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (iPad; CPU OS 17_0_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0.1 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 15_7_9 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.6.5 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) FxiOS/118.0 Mobile/15E148 Safari/605.1.15",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 16_6_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0.1 Mobile/15E148 Safari/604.1",
}

// colors
const (
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorReset  = "\033[0m"
	ColorBold   = "\033[1m"
	ColorDim    = "\033[2m"
	ColorUnder  = "\033[4m"
	ColorBlink  = "\033[5m"
	ColorRev    = "\033[7m"
	ColorHidden = "\033[8m"
)

// token bucket
var rateLimiter chan struct{}

// product names to ignore (case-insensitive)
var FORBIDDEN_GOODS = []string{
	"alverde",
	"bambucké",
	"do myčky",
	"doplněk stravy",
	"express menu",
	"filtr",
	"formičky",
	"gimcat",
	"holení",
	"hotovky",
	"inspirace",
	"jídelní set",
	"kartáček",
	"kolekce",
	"kolínská",
	"konkor",
	"koupele",
	"krku",
	"kráječ",
	"křeslo",
	"lepidlo",
	"mast",
	"matrace",
	"micelární",
	"motorový",
	"měděná",
	"na vlasy",
	"nosní",
	"obývací stěna",
	"okrasná",
	"pamlsky",
	"parfemovaná",
	"parfém",
	"pleť",
	"pleťová",
	"postel",
	"razítko",
	"rostoucí vejce",
	"rty",
	"severochema",
	"sklenice",
	"společenská hra",
	"stojánková baterie",
	"sůl koupelová",
	"tablety",
	"tescoma",
	"toaletní",
	"tělo",
	"vitamín",
	"vlasová voda",
	"vonné tyčinky",
	"vykrajovátka",
	"zdravá zahrada",
	"zubní",
	"zuby",
	"úklid",
	"ústní",
	"škrobenka",
	"šťouchadlo",
}

// product structure
type Goods struct {
	Category     string
	Query        string
	Name         string
	Price        string
	PricePerUnit string
	Discount     string
	Note         string
	Club         string
	Volume       string
	Market       string
	Validity     string
	Url          string
	ImageUrl     string
	SubCat       string
}

var CZreplacer = strings.NewReplacer(
	"á", "a",
	"č", "c",
	"ď", "d",
	"é", "e",
	"ě", "e",
	"í", "i",
	"ľ", "l",
	"ň", "n",
	"ó", "o",
	"ř", "r",
	"š", "s",
	"ť", "t",
	"ú", "u",
	"ů", "u",
	"ý", "y",
	"ž", "z",
)

var nonAlphanumeric = regexp.MustCompile("[^a-z0-9]+")

// normalization - helper function to normalize Czech strings for comparison
func normalization(s string) string {
	// 1. malá písmena
	s = strings.ToLower(s)
	// 2. odstranění diakritiky
	s = CZreplacer.Replace(s)
	// 3. nahrazení nealfanumerických znaků mezerou
	s = nonAlphanumeric.ReplaceAllString(s, " ")
	// 4. ořez white space
	s = strings.TrimSpace(s)
	return s
}

// deduplicateGoods
func deduplicateGoods(scrapedGoods []Goods) []Goods {
	uniqueGoodsMap := make(map[string]Goods)
	for _, good := range scrapedGoods {
		normalizedNote := normalization(good.Note)
		key := good.Name +
			good.Price +
			good.PricePerUnit +
			normalizedNote + // apply normalization
			good.Club +
			good.Volume +
			good.Market +
			good.Validity
		uniqueGoodsMap[key] = good
	}
	var finalGoods []Goods
	for _, good := range uniqueGoodsMap {
		finalGoods = append(finalGoods, good)
	}
	return finalGoods
}

// check the lock
func CheckLock() bool {
	pid := os.Getpid()

	// 1. read the lock
	content, err := os.ReadFile(LOCK_FILE)
	if err == nil {
		fileInfo, _ := os.Stat(LOCK_FILE)

		// A. check lock age
		if time.Since(fileInfo.ModTime()) > LOCK_FILE_DURATION {
			// Soubor je starší než LOCK_DURATION (1 hodina) -> Předpokládáme Zombie Lock. Smažeme jej a vytvoříme nový.
			fmt.Printf("🔒 Lock file %s found but is too old (modified %s). Deleting old lock.\n", LOCK_FILE, fileInfo.ModTime().Format(time.RFC3339))
			if err := os.Remove(LOCK_FILE); err != nil {
				fmt.Printf("🚨 ERROR: failed to remove old lock file: %v\n", err)
				return false
			}
		} else {
			// B. lock is new - check the content
			lockedPID, parseErr := strconv.Atoi(string(content))
			if parseErr == nil && isProcessRunning(lockedPID) {
				if lockedPID == pid {
					// lock is ours - theoretical situation
					fmt.Printf("⚠️ WARNING: lock file %s exists and contains current PID. Proceeding.\n", LOCK_FILE)
					return true
				}
				// lock is not ours
				fmt.Printf("❌ ABORT: lock file %s found for active PID %d. Run aborted.\n", LOCK_FILE, lockedPID)
				return false
			}
			// C. lock exists, but is invalid
			fmt.Printf("⚠️ WARNING: lock file %s exists but PID %d not running (or invalid). Overwriting.\n", LOCK_FILE, lockedPID)
		}
	}

	// 2. make a new lock
	fmt.Printf("✅ Creating new lock file %s with PID %d.\n", LOCK_FILE, pid)
	if err := os.WriteFile(LOCK_FILE, []byte(strconv.Itoa(pid)), 0644); err != nil {
		fmt.Printf("🚨 ERROR: failed to create lock file: %v\n", err)
		return false
	}
	return true
}

// unlock the lock
func Unlock() {
	pid := os.Getpid()
	content, err := os.ReadFile(LOCK_FILE)
	if err == nil && strconv.Itoa(pid) == string(content) {
		if err := os.Remove(LOCK_FILE); err != nil {
			fmt.Printf("🚨 ERROR: failed to remove lock file %s: %v\n", LOCK_FILE, err)
		} else {
			fmt.Printf("🔓 Lock file %s removed.\n", LOCK_FILE)
		}
	} else if err != nil && !os.IsNotExist(err) {
		fmt.Printf("🚨 ERROR: failed to read lock file for verification: %v\n", err)
	} else {
		fmt.Printf("⚠️ WARNING: could not verify/remove lock file %s (file not found or content mismatch).\n", LOCK_FILE)
	}
}

// isProcessRunning - helper function for process existence
func isProcessRunning(pid int) bool {
	return syscall.Kill(pid, syscall.Signal(0)) == nil
}

// isForbidden- helper function to check if product name contains forbidden strings
func isForbidden(name string, forbidden []string) bool {
	lowerName := strings.ToLower(name)
	for _, s := range forbidden {
		if strings.Contains(lowerName, strings.ToLower(s)) {
			return true
		}
	}
	return false
}

// extractGoodsFromHtml - extract data from HTML
func extractGoodsFromHtml(doc *goquery.Document, category string, query string) []Goods {
	var goods []Goods
	doc.Find("div.group_discounts").Each(func(i int, s *goquery.Selection) {
		// ignore .notactive
		if s.HasClass("notactive") {
			return
		}

		// extract general product info once per group
		nameSelection := s.Find("div.product_name h2 a")
		productName := strings.TrimSpace(nameSelection.Text())
		productName = sanitizeString(productName)

		// skip forbidden goods
		if isForbidden(productName, FORBIDDEN_GOODS) {
			return
		}

		productUrl, _ := nameSelection.Attr("href")
		if !strings.HasPrefix(productUrl, "http") {
			productUrl = KOOPI_HOME_URL + productUrl
		}

		imgSelection := s.Find("div.product_image a img")
		productImageUrl, _ := imgSelection.Attr("data-src")
		if !strings.HasPrefix(productImageUrl, "http") {
			productImageUrl = KOOPI_IMAGE_URL + productImageUrl
		}

		// iterate through each specific offer within the product group
		s.Find(".discount_row").Each(func(j int, offer *goquery.Selection) {
			var newGoods Goods
			newGoods.Category = category
			newGoods.Query = query
			newGoods.Name = productName
			newGoods.Url = productUrl
			newGoods.ImageUrl = productImageUrl

			// price
			newGoods.Price = strings.TrimSpace(offer.Find(".discount_price_value").Text())
			newGoods.Price = strings.ReplaceAll(newGoods.Price, ",", ".")

			// price per unit
			newGoods.PricePerUnit = strings.TrimSpace(offer.Find(".price_per_unit").Text())
			newGoods.PricePerUnit = strings.ReplaceAll(newGoods.PricePerUnit, ",", ".")

			// discount
			newGoods.Discount = strings.TrimSpace(offer.Find(".discount_percentage").Text())
			newGoods.Discount = strings.ReplaceAll(newGoods.Discount, "–", "-")
			newGoods.Discount = strings.TrimSpace(newGoods.Discount)
			//newGoods.Discount = strings.TrimPrefix(newGoods.Discount, "–")
			//newGoods.Discount = strings.TrimSuffix(newGoods.Discount, "%")

			// volume
			newGoods.Volume = strings.TrimSpace(offer.Find(".discount_amount").Text())
			newGoods.Volume = strings.TrimPrefix(newGoods.Volume, "/")
			newGoods.Volume = strings.TrimSpace(newGoods.Volume)

			// note
			newGoods.Note = strings.TrimSpace(offer.Find(".discount_note").Text())
			newGoods.Note = strings.ReplaceAll(newGoods.Note, "+3 Kč záloha na láhev", "zálohovaná lahev")
			newGoods.Note = strings.ReplaceAll(newGoods.Note, "láhev", "lahev")
			newGoods.Note = strings.ReplaceAll(newGoods.Note, "láhve", "lahve")
			newGoods.Note = strings.ReplaceAll(newGoods.Note, "vybrané druhy", "různé druhy")
			newGoods.Note = sanitizeString(newGoods.Note)

			// club
			newGoods.Club = strings.TrimSpace(offer.Find(".discounts_club").Text())
			newGoods.Club = sanitizeString(newGoods.Club)

			// validity
			newGoods.Validity = strings.TrimSpace(offer.Find(".discounts_validity").Text())
			newGoods.Validity = sanitizeString(newGoods.Validity)

			// market
			newGoods.Market = strings.TrimSpace(offer.Find(".discounts_shop_name a span").Text())
			newGoods.Market = sanitizeString(newGoods.Market)

			// add SubCat based on Note
			if strings.Contains(newGoods.Note, "zálohovaná lahev") {
				newGoods.SubCat = "lahev"
			}
			if strings.Contains(newGoods.Note, "plech") {
				newGoods.SubCat = "plech"
			}

			if newGoods.Name != "" {
				// append the struct to the global list
				goods = append(goods, newGoods)
			}
		})
	})
	return goods
}

// saveToCache - save HTML to cache
func saveToCache(cacheName string, content []byte) {
	if _, err := os.Stat(HTML_CACHE); os.IsNotExist(err) {
		err = os.MkdirAll(HTML_CACHE, 0755)
		if err != nil {
			log.Printf("[%s] 💥 error creating cache folder [%s]: %v", cacheName, HTML_CACHE, err)
			return
		}
	}
	filePath := filepath.Join(HTML_CACHE, cacheName)
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		log.Printf("[%s] 💥 error saving to cache: %v", cacheName, err)
	}
}

// loadFromCache - load HTML from cache
func loadFromCache(cacheName string) (*goquery.Document, error) {
	filePath := filepath.Join(HTML_CACHE, cacheName)
	localFileContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(localFileContent)))
	if err != nil {
		log.Printf("[%s] 😵‍💫 error creating document from cache: %v", cacheName, err)
		return nil, err
	}
	return doc, nil
}

// saveImageToCache - save image to cache for WebP processing
func saveImageToCache(imageUrl string) {
	if _, err := os.Stat(IMAGE_CACHE); os.IsNotExist(err) {
		err = os.MkdirAll(IMAGE_CACHE, 0755)
		if err != nil {
			log.Printf("[%s] 💥 error creating image cache folder: %v", IMAGE_CACHE, err)
			return
		}
	}

	fileName := filepath.Base(imageUrl)
	filePath := filepath.Join(IMAGE_CACHE, fileName)
	if _, err := os.Stat(filePath); err == nil {
		return
	}

	log.Printf("📥 downloading %s%s%s", ColorCyan, imageUrl, ColorReset)

	resp, err := http.Get(imageUrl)
	if err != nil {
		log.Printf("[%s] 💥 error downloading image: %v", imageUrl, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Printf("[%s] 💥 failed to download image, code: %d", imageUrl, resp.StatusCode)
		return
	}
	file, err := os.Create(filePath)
	if err != nil {
		log.Printf("[%s] 💥 error creating file for image: %v", fileName, err)
		return
	}
	defer file.Close()
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		log.Printf("[%s] 💥 error saving image to file: %v", fileName, err)
	}
}

// scrapePage - scrape pages (cache/online)
func scrapePage(UA string, ctx context.Context, urlToScrape string, cacheName string, category string, query string, allGoods *[]Goods, mutex *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()

	// 1. try cache first
	doc, err := loadFromCache(cacheName)
	if err == nil {
		goodsList := extractGoodsFromHtml(doc, category, query)
		mutex.Lock()
		for _, good := range goodsList {
			saveImageToCache(good.ImageUrl)
		}
		*allGoods = append(*allGoods, goodsList...)
		mutex.Unlock()

		// console stats
		if len(goodsList) == 0 {
			log.Printf("📦 %d [%s] %sextracted 0 items (cache) %s%s%s", len(*allGoods), query, ColorRed, ColorCyan, urlToScrape, ColorReset)
		} else {
			log.Printf("📦 %d [%s] extracted %s%d items%s (cache)", len(*allGoods), query, ColorBlue, len(goodsList), ColorReset)
		}
		return
	}

	// 2. Rate Limiter Acquisition (Only for network scrape)
	select {
	case <-ctx.Done():
		// Task cancelled before acquiring token
		return
	case <-rateLimiter:
		defer func() {
			// A. Calculate sleep time
			sleepTime := time.Duration(rand.Intn(20000)+7000) * time.Millisecond

			// B. Wait on a Timer or Context Done (INTERRUPTIBLE SLEEP!)
			timer := time.NewTimer(sleepTime)

			select {
			case <-ctx.Done():
				timer.Stop()
				log.Printf("❌ [%s] sleep interrupted", query)
			case <-timer.C:
				// Timer finished normally.
			}

			// C. Return the token.
			rateLimiter <- struct{}{}
		}()
	}

	log.Printf("🔎 [%s] scrape %s%s%s", query, ColorCyan, urlToScrape, ColorReset)

	client := &http.Client{
		Timeout: REQ_TIMEOUT,
	}
	req, err := http.NewRequestWithContext(ctx, "GET", urlToScrape, nil)
	if err != nil {
		log.Printf("[%s] 💥 error in request: %v", query, err)
		return
	}
	req.Header.Set("User-Agent", UA)
	res, err := client.Do(req)
	if err != nil {
		//		log.Printf("[%s] 💥 error during request: %v", query, err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Printf("[%s] 💥 request code [%d]: '%s'", query, res.StatusCode, res.Status)
		return
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("[%s] 💥 error reading response body: %v", query, err)
		return
	}
	resDoc, err := goquery.NewDocumentFromReader(bytes.NewReader(bodyBytes))
	if err != nil {
		log.Printf("[%s] 😵‍💫 error creating document: %v", query, err)
		return
	}

	// extract goods from HTML
	goodsList := extractGoodsFromHtml(resDoc, category, query)

	// save HTML to cache
	saveToCache(cacheName, bodyBytes)

	// extract goods images
	mutex.Lock()
	for _, good := range goodsList {
		saveImageToCache(good.ImageUrl)
	}
	*allGoods = append(*allGoods, goodsList...)
	total := len(*allGoods)
	mutex.Unlock()

	// console
	if total == 0 {
		log.Printf("📦 %d [%s] %sextracted 0 items %s%s%s", total, query, ColorRed, ColorCyan, urlToScrape, ColorReset)
		return
	} else {
		log.Printf("📦 %d [%s] extracted %s%d items%s", total, query, ColorBlue, len(goodsList), ColorReset)
	}
}

// sanitizeString - remove spaces and newlines
func sanitizeString(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

// appendToCsv - add data to CSV
func appendToCsv(goods []Goods, filename string, mutex *sync.Mutex) {
	mutex.Lock()
	defer mutex.Unlock()

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("[%s] 💥 error opening for writing: %v", filename, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Comma = ';'
	headers := []string{"Name", "Price", "PricePerUnit", "Discount", "Category", "SubCat", "Note", "Club", "Volume", "Market", "Validity", "Url", "ImageUrl", "Query"}
	writer.Write(headers)

	for _, item := range goods {
		item.ImageUrl = strings.TrimPrefix(item.ImageUrl, "https://img.kupi.cz/kupi/thumbs/")
		item.ImageUrl = strings.TrimPrefix(item.ImageUrl, "https://img.kupi.cz/img/no_img/no_discounts.png")
		item.ImageUrl = strings.TrimPrefix(item.ImageUrl, "https://img.kupi.cz/")

		cleanUrl := strings.TrimPrefix(item.Url, KOOPI_HOME_URL)
		writer.Write([]string{
			item.Name,
			item.Price,
			item.PricePerUnit,
			item.Discount,
			item.Category,
			item.SubCat,
			item.Note,
			item.Club,
			item.Volume,
			item.Market,
			item.Validity,
			cleanUrl,
			item.ImageUrl,
			item.Query,
		})
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		log.Fatalf("[%s] 💥 error writing: %v", filename, err)
	}
}

// appendToJson - save data to JSON
func appendToJson(goods []Goods, filename string, markets []string, mutex *sync.Mutex) {
	mutex.Lock()
	defer mutex.Unlock()

	// this map is used to find how many offers exist for a given product name/volume combination
	genericProductCounts := make(map[string]int)
	for _, item := range goods {
		genericHashKey := item.Name + item.Volume + item.Category + item.SubCat
		genericProductCounts[genericHashKey]++
	}

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("[%s] 💥 error opening for writing: %v", filename, err)
	}
	defer file.Close()

	var cleanedGoods []map[string]any
	for _, item := range goods {
		hashString := item.Name + item.Volume + item.Category + item.SubCat // unique good hash (for ID)
		hash := md5.Sum([]byte(hashString))
		md5Hash := hex.EncodeToString(hash[:])

		// retrieve the offer count for the generic product
		genericHashKey := item.Name + item.Volume + item.Category + item.SubCat
		offerCount := genericProductCounts[genericHashKey]

		cleanedItem := make(map[string]any)
		cleanedItem["id"] = md5Hash
		cleanedItem["cat"] = item.Category
		cleanedItem["subcat"] = item.SubCat
		cleanedItem["query"] = item.Query
		cleanedItem["name"] = item.Name
		cleanedItem["price"] = strings.Replace(item.Price, ",", ".", 1)
		cleanedItem["priceperunit"] = item.PricePerUnit
		cleanedItem["discount"] = item.Discount
		cleanedItem["note"] = item.Note
		cleanedItem["club"] = item.Club
		cleanedItem["volume"] = item.Volume
		cleanedItem["market"] = item.Market
		cleanedItem["validity"] = item.Validity
		cleanedItem["url"] = strings.TrimPrefix(item.Url, KOOPI_HOME_URL)

		imageURL := item.ImageUrl
		if before, ok := strings.CutSuffix(imageURL, ".png"); ok {
			imageURL = before + ".webp"
		} else if before0, ok0 := strings.CutSuffix(imageURL, ".jpg"); ok0 {
			imageURL = before0 + ".webp"
		}
		imageURL = strings.TrimPrefix(imageURL, "https://img.kupi.cz/kupi/thumbs/")
		imageURL = strings.TrimPrefix(imageURL, "https://img.kupi.cz/img/no_img/no_discounts.png")
		if imageURL == "" || strings.Contains(imageURL, "no_discounts") {
			imageURL = "default.png"
		}
		cleanedItem["image"] = imageURL
		cleanedItem["offer_count"] = offerCount

		cleanedGoods = append(cleanedGoods, cleanedItem)
	}

	outputData := make(map[string]any)
	outputData["created"] = time.Now().Format(time.RFC3339)
	outputData["count"] = len(cleanedGoods)
	outputData["goods"] = cleanedGoods
	outputData["markets"] = markets
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(outputData); err != nil {
		log.Fatalf("[%s] 💥 error writing to JSON: %v", filename, err)
	}
}

// MAIN * MAIN * MAIN
func main() {
	if !CheckLock() {
		os.Exit(1)
	}
	defer Unlock()

	//log.SetFlags(log.Ltime | log.Lshortfile)
	log.SetFlags(0)

	// set random UA
	UA := UserAgents[rand.Intn(len(UserAgents))]
	log.Printf("User Agent: %s", UA)
	time.Sleep(2 * time.Second)

	// rate limiter
	rateLimiter = make(chan struct{}, MAX_THREADS)
	for range MAX_THREADS {
		rateLimiter <- struct{}{}
	}

	// load input CSV
	file, err := os.Open(INPUT_CSV)
	if err != nil {
		log.Fatalf("[%s] 💥 error opening: %v", INPUT_CSV, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ','
	inputRecords, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("[%s] 💥 error reading: %v", INPUT_CSV, err)
	}

	if len(inputRecords) == 0 {
		log.Printf("😐️ [%s] is empty. Nothing to scrape.", INPUT_CSV)
		return
	}

	var urlsToScrape []struct {
		url      string
		cacheKey string
		category string
		query    string
	}

	// generate URLs to scrape
	for _, record := range inputRecords {
		category := strings.TrimSpace(record[0])
		query := strings.TrimSpace(record[1])
		pages, _ := strconv.Atoi(strings.TrimSpace(record[2]))
		escapedQuery := url.QueryEscape(query)
		for pageNum := 1; pageNum <= pages; pageNum++ {
			var urlStr string
			if pageNum == 1 {
				urlStr = KOOPI_SEARCH_URL + escapedQuery
			} else {
				urlStr = fmt.Sprintf("%s%s%s%d", KOOPI_SEARCH_URL, escapedQuery, KOOPI_SUBPAGE, pageNum)
			}
			cacheKey := fmt.Sprintf("%s-%d.html", strings.ReplaceAll(query, " ", "-"), pageNum)

			urlsToScrape = append(urlsToScrape, struct {
				url      string
				cacheKey string
				category string
				query    string
			}{urlStr, cacheKey, category, query})
		}
	}

	// shuffle URLs
	rand.Shuffle(len(urlsToScrape), func(i, j int) {
		urlsToScrape[i], urlsToScrape[j] = urlsToScrape[j], urlsToScrape[i]
	})

	// limits
	if len(urlsToScrape) == 0 {
		log.Println("🍀 Nothing to scrape.")
		return
	}
	if len(urlsToScrape) > MAX_SCRAPED_GOODS {
		urlsToScrape = urlsToScrape[:MAX_SCRAPED_GOODS]
	}

	var newScrapedGoods []Goods
	var wg sync.WaitGroup
	var csvMutex sync.Mutex
	var goodsMutex sync.Mutex

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// signals handling
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signals
		log.Println("\n\nInterrupted ...")
		cancel()
	}()

	// concurrency
	concurrencyLimit := make(chan struct{}, MAX_THREADS)

	// workers
	for _, urlData := range urlsToScrape {
		wg.Add(1)
		concurrencyLimit <- struct{}{}
		go func(urlData struct {
			url      string
			cacheKey string
			category string
			query    string
		}) {
			defer func() {
				<-concurrencyLimit
			}()
			scrapePage(UA, ctx, urlData.url, urlData.cacheKey, urlData.category, urlData.query, &newScrapedGoods, &goodsMutex, &wg)
		}(urlData)
	}

	// wait for workers to finish
	wg.Wait()

	// deduplication
	finalGoods := deduplicateGoods(newScrapedGoods)

	// create stats
	uniqueMarkets := make(map[string]struct{})
	marketCounts := make(map[string]int)
	uniqueVolumes := make(map[string]struct{})

	for _, good := range finalGoods {
		if good.Market != "" {
			uniqueMarkets[good.Market] = struct{}{}
			marketCounts[good.Market]++
		}
		if good.Volume != "" {
			uniqueVolumes[good.Volume] = struct{}{}
		}
	}
	var marketsList []string
	for market := range uniqueMarkets {
		marketsList = append(marketsList, market)
	}
	sort.Strings(marketsList)

	var marketStatsList []string
	for _, market := range marketsList {
		marketStatsList = append(marketStatsList, fmt.Sprintf("%s (%d)", market, marketCounts[market]))
	}

	var volumesList []string
	for volume := range uniqueVolumes {
		volumesList = append(volumesList, volume)
	}
	sort.Strings(volumesList)

	// console printouts
	fmt.Println("\n🏪 Markets:", strings.Join(marketStatsList, ", "))
	//fmt.Println("\n🥡 Volumes:", strings.Join(volumesList, ", "))

	// Czech sorting
	c := collate.New(language.Czech)
	sort.Slice(finalGoods, func(i, j int) bool {
		return c.CompareString(finalGoods[i].Name, finalGoods[j].Name) < 0
	})

	// save sorted data to CSV
	appendToCsv(finalGoods, OUTPUT_CSV, &csvMutex)

	// save sorted data to JSON
	appendToJson(finalGoods, OUTPUT_JSON, marketsList, &csvMutex)

	fmt.Printf("\n🍀 Scraping finished %d unique items.\n", len(finalGoods))
}

package movers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"
	"unicode"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/text/transform"
)

// MoverList is a specific list of securities with significant price movements published by the WSJ
type MoverList string

const (
	// USCompositeGainers is a list of the stocks with the largest percentage gains across NYSE, NASDAQ and Arca.
	// Includes common, closed end funds, ETFs, ETNs and REITS with prior day close of $2 a share or higher, with volume of at least 2,000.
	USCompositeGainers MoverList = `http://www.wsj.com/mdc/public/page/2_3021-gaincomp-gainer-%d%02d%02d.html?mod=mdc_pastcalendar`
	// USCompositeLosers is a list of the stocks with the largest percentage decline across NYSE, NASDAQ and Arca.
	// Includes common, closed end funds, ETFs, ETNs and REITS with prior day close of $2 a share or higher, with volume of at least 2,000.
	USCompositeLosers MoverList = `http://www.wsj.com/mdc/public/page/2_3021-losecomp-loser-%d%02d%02d.html?mod=mdc_pastcalendar`
)

var (
	nameAndSymbolPattern = regexp.MustCompile(`(.+)\s\((.+)\)`)
	netClient            = &http.Client{
		Timeout: time.Second * 5,
	}
)

// Stock represents an equity security listing that experienced a price movement
type Stock struct {
	Ticker    string  `json:"ticker"`
	Name      string  `json:"instrument"`
	Price     float64 `json:"price"`
	PctChange float64 `json:"percentChange"`
	Volume    int     `json:"volume"`
}

// Date represents a calendar date
type Date struct {
	Year  int
	Month time.Month
	Day   int
}

// NewDate returns a Date value suitable for passing to a Getter
func NewDate(year int, month time.Month, day int) (d Date, err error) {
	d = Date{
		Year:  year,
		Month: month,
		Day:   day,
	}
	err = d.Validate()
	return
}

// Validate returns an error if the date is not a weekday after 12/31/2009
func (d *Date) Validate() error {
	if d.Year > time.Now().Year() || d.Year < 2010 {
		return errors.New(`invalid year`)
	}
	if d.Day < 1 || d.Day > 31 {
		return errors.New(`invalid day`)
	}
	t := time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC)
	switch t.Weekday() {
	case time.Saturday, time.Sunday:
		return fmt.Errorf(`Movers data is not available on weekends`)
	}
	return nil
}

// Getter is implemented by things that allow fetching of stock mover lists by date
type Getter interface {
	Get(list MoverList, d Date) (result []Stock, err error)
}

// NewGetter returns a Getter backed by an in-memory cache
func NewGetter() Getter {
	return &cache{
		data: make(map[string]*cached),
	}
}

type cached struct {
	sync.Mutex
	data []Stock
}

type cache struct {
	sync.Mutex
	data map[string]*cached
}

func dataURL(list MoverList, d Date) (url string, err error) {
	if err = d.Validate(); err != nil {
		return
	}
	return fmt.Sprintf(string(list), d.Year, int(d.Month), d.Day), nil
}

func (c *cache) Get(list MoverList, d Date) (results []Stock, err error) {
	c.Lock()
	var addr string
	addr, err = dataURL(list, d)
	if err != nil {
		c.Unlock()
		return
	}
	val, ok := c.data[addr]
	if !ok {
		val = &cached{}
		c.data[addr] = val
	}
	val.Lock()
	c.Unlock()
	defer val.Unlock()
	if len(val.data) > 0 {
		results = val.data
		return
	}
	var res *http.Response
	res, err = netClient.Get(addr)
	if err != nil {
		return
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		err = fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
		return
	}
	return parseTable(res.Body)
}

func parseTable(data io.Reader) (results []Stock, err error) {
	var doc *goquery.Document
	doc, err = goquery.NewDocumentFromReader(data)
	if err != nil {
		return
	}
	doc.Find(`table.mdcTable tbody tr`).Slice(1, goquery.ToEnd).EachWithBreak(func(i int, sel *goquery.Selection) bool {
		var s Stock
		s, err = trToStock(sel)
		if err != nil {
			return false
		}
		results = append(results, s)
		return true
	})
	return
}

func trToStock(sel *goquery.Selection) (s Stock, err error) {
	data := sel.Find("td").Map(func(i int, sel2 *goquery.Selection) string {
		switch i {
		case 1:
			return sel2.Text()
		default:
			t := sel2.Text()
			b := make([]byte, len(t))
			n, _, _ := filterNonNumeric.Transform(b, []byte(t), true)
			return string(b[:n])
		}
	})
	if len(data) != 6 {
		err = fmt.Errorf(`expected 6 columns, got %d`, len(data))
		return
	}
	matches := nameAndSymbolPattern.FindStringSubmatch(data[1])
	if len(matches) != 3 {
		err = fmt.Errorf(`expected stock name and ticker regex match to have 3 elements, got %d`, len(matches))
		return
	}
	s.Name = matches[1]
	s.Ticker = matches[2]
	s.Volume, err = strconv.Atoi(data[5])
	if err != nil {
		return
	}
	s.Price, err = strconv.ParseFloat(data[2], 64)
	if err != nil {
		return
	}
	s.PctChange, err = strconv.ParseFloat(data[4], 64)
	return
}

var filterNonNumeric = transform.RemoveFunc(func(r rune) bool {
	switch r {
	case '-', '.':
		return false
	}
	return !unicode.IsNumber(r)
})

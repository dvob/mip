package mip

import (
	"encoding/json"
	"fmt"
	"github.com/ghodss/yaml"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"io"
	"io/ioutil"
	"strings"
	"time"
)

type Importer interface {
	Name() string
	Run() (*ImportSummary, error)
}

type Export struct {
	w      io.Writer
	offset int64
}

var Encodings = map[string]encoding.Encoding{
	"iso-8859-1": charmap.ISO8859_1,
}

func NewExport(output io.Writer, enc string) (*Export, error) {
	// no conversion needed
	if enc == "utf8" || enc == "" {
		return &Export{output, 0}, nil
	}

	targetEnc, ok := Encodings[enc]
	if !ok {
		return &Export{}, fmt.Errorf("unknown encoding '%s'", enc)
	}
	output = encoding.HTMLEscapeUnsupported(targetEnc.NewEncoder()).Writer(output)
	return &Export{output, 0}, nil
}

func (e *Export) Write(p []byte) (n int, err error) {
	bytes := 0
	if e.offset == 0 {
		io.WriteString(e.w, FormatHeader())
		bytes += len(FormatHeader())
	}
	i, err := e.w.Write(p)
	bytes += i
	e.offset += int64(bytes)
	return bytes, err
}

type Record struct {
	Id             string
	IdPrefix       string
	Description    string
	PurchasePrice  float64
	PurchaseFactor float64
	SellingFactor  float64
	SellingPrice   float64
	Category       string
	CategoryNumber string
}

func (r *Record) FormatLine() string {
	replacer := strings.NewReplacer(",", " ", "\"", "\"\"")
	return fmt.Sprintf("\"%s%s\",\"%s\",\"%s\",\"%.2f\",\"%f\",\"%.2f\",\"%f\",\"%s\"\n",
		r.IdPrefix,
		r.Id,
		replacer.Replace(r.Description),
		r.Category,
		r.PurchasePrice,
		r.PurchaseFactor,
		r.SellingPrice,
		r.SellingFactor,
		r.CategoryNumber)
}

func FormatHeader() string {
	return "\"Id\",\"Beschreibung\",\"Kategorie\",\"Einkaufspreis\",\"Einkaufsfaktor\",\"Verkaufspreis\",\"Verkaufsfaktor\",\"Kategorie-Nummer\"\n"
}

type Config struct {
	OutputFile     string        `json:"output_file"`
	OutputEncoding string        `json:"output_encoding"`
	AlltronConfig  AlltronConfig `json:"alltron"`
	SupragConfig   SupragConfig  `json:"suprag"`
	MitelConfig    MitelConfig   `json:"mitel"`
}

func ReadJsonConfig(path string, config *Config) error {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	err = json.Unmarshal(bytes, config)
	if err != nil {
		return err
	}
	return nil
}

func ReadYamlConfig(path string, config *Config) error {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(bytes, config)
	if err != nil {
		return err
	}
	return nil
}

type ImportSummary struct {
	start    time.Time
	end      time.Time
	Articles int
	Ignored  int
}

func NewImportSummary() *ImportSummary {
	return &ImportSummary{}
}

func StartImportSummary() *ImportSummary {
	ps := &ImportSummary{}
	ps.Start()
	return ps
}

func (ps *ImportSummary) Add(a *ImportSummary) {
	ps.Articles += a.Articles
	ps.Ignored += a.Ignored
}

func (ps *ImportSummary) String() string {
	return fmt.Sprintf("processed %d articles in %s (ignored : %d)", ps.Articles, ps.Duration(), ps.Ignored)
}

func (ps *ImportSummary) Start() {
	ps.start = time.Now()
}

func (ps *ImportSummary) Stop() {
	ps.end = time.Now()
}

func (ps *ImportSummary) Duration() time.Duration {
	if ps.start.IsZero() {
		return time.Duration(0)
	}
	if ps.end.IsZero() {
		return time.Since(ps.start)
	}

	return ps.end.Sub(ps.start)
}

package mip

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"github.com/dsbrng25b/mip/ftp"
	"github.com/spf13/viper"
	"gopkg.in/cheggaaa/pb.v1"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

type AlltronConfig struct {
	ArticleFile       string              `json:"article_file"`
	PriceFile         string              `json:"price_file"`
	IgnoredCategories map[string][]string `json:"ignored"`
	SellingFactor     float64             `json:"selling_factor"`
	PurchaseFactor    float64             `json:"purchase_factor"`
	IdPrefix          string              `json:"id_prefix"`
	Category          string              `json:"category"`
	CategoryNumber    string              `json:"category_number"`
	ShowProgress      bool                `json:"show_progress"`
	UseFtp            bool                `json:"use_ftp"`
	FtpUser           string              `json:"ftp_user"`
	FtpPassword       string              `json:"ftp_password"`
	FtpAddress        string              `json:"ftp_address"`
	FtpArticleFile    string              `json:"ftp_article_file"`
	FtpPriceFile      string              `json:"ftp_price_file"`
	FtpSaveFiles      bool                `json:"ftp_save_files"`
	FtpSaveDir        string              `json:"ftp_save_dir"`
}

type AlltronImport struct {
	name    string
	cfg     *viper.Viper
	summary *ImportSummary
	output  io.Writer
	prices  map[string]XmlArticlePrice
	bar     *pb.ProgressBar
}

func NewAlltronImport(cfg *viper.Viper, output io.Writer) *AlltronImport {
	a := &AlltronImport{
		name:    "Alltron",
		cfg:     cfg,
		summary: NewImportSummary(),
		output:  output,
		prices:  make(map[string]XmlArticlePrice),
	}

	if cfg.GetBool("show_progress") {
		a.bar = pb.New64(0).SetUnits(pb.U_BYTES)
	}
	return a
}

func (i *AlltronImport) Name() string {
	return i.name
}

type XmlArticlePrice struct {
	Id            string  `xml:"LITM"`
	PurchasePrice float64 `xml:"price>EXPR"`
}

type XmlArticle struct {
	Id          string `xml:"LITM"`
	Description string `xml:"part_description>DESC"`
	Maft        string `xml:"additional_information>MAFT"`
	Cat1        string `xml:"part_catagory>CAT1"`
}

func (i *AlltronImport) getFtpReaders() (ar, pr io.ReadCloser, err error) {
	articleReader, err := ftp.Open(
		i.cfg.GetString("ftp_address"),
		i.cfg.GetString("ftp_user"),
		i.cfg.GetString("ftp_password"),
		i.cfg.GetString("ftp_article_file"))
	if err != nil {
		return nil, nil, err
	}
	priceReader, err := ftp.Open(
		i.cfg.GetString("ftp_address"),
		i.cfg.GetString("ftp_user"),
		i.cfg.GetString("ftp_password"),
		i.cfg.GetString("ftp_price_file"))
	if err != nil {
		return nil, nil, err
	}

	if i.bar != nil {
		i.bar.Total = articleReader.Size
	}
	return articleReader, priceReader, nil
}

func (i *AlltronImport) getFileReaders() (ar, pr io.ReadCloser, err error) {
	articleFile, err := os.Open(i.cfg.GetString("article_file"))
	if err != nil {
		return nil, nil, err
	}

	priceFile, err := os.Open(i.cfg.GetString("price_file"))
	if err != nil {
		return nil, nil, err
	}
	if i.bar != nil {
		fi, err := articleFile.Stat()
		if err != nil {
			return nil, nil, err
		}
		i.bar.Total = fi.Size()
	}

	return articleFile, priceFile, nil
}

func (i *AlltronImport) Run() (*ImportSummary, error) {

	var articleFtpSave io.WriteCloser
	var priceFtpSave io.WriteCloser
	var articleReader io.ReadCloser
	var priceReader io.ReadCloser
	var err error

	if i.cfg.GetBool("use_ftp") {
		articleReader, priceReader, err = i.getFtpReaders()
	} else {
		articleReader, priceReader, err = i.getFileReaders()
	}
	if err != nil {
		return i.summary, err
	}
	defer articleReader.Close()
	defer priceReader.Close()

	if i.cfg.GetBool("show_progress") {
		articleReader = i.bar.NewProxyReader(articleReader)
	}

	if i.cfg.GetBool("use_ftp") && i.cfg.GetBool("ftp_save_files") {
		articleFtpSave, err = os.Create(
			filepath.Join(
				i.cfg.GetString("ftp_save_dir"),
				filepath.Base(i.cfg.GetString("ftp_article_file"))))
		if err != nil {
			return i.summary, err
		}
		defer articleFtpSave.Close()
		articleReader = ioutil.NopCloser(io.TeeReader(articleReader, articleFtpSave))

		priceFtpSave, err = os.Create(
			filepath.Join(
				i.cfg.GetString("ftp_save_dir"),
				filepath.Base(i.cfg.GetString("ftp_price_file"))))
		if err != nil {
			return i.summary, err
		}
		defer priceFtpSave.Close()
		priceReader = ioutil.NopCloser(io.TeeReader(priceReader, priceFtpSave))
	}

	if i.cfg.GetBool("show_progress") {
		i.bar.Start()
	}
	_, err = i.process(articleReader, priceReader)
	if i.cfg.GetBool("show_progress") {
		i.bar.Finish()
	}
	if err != nil {
		return i.summary, err
	}

	return i.summary, nil

}

func (i *AlltronImport) process(articleReader, priceReader io.Reader) (*ImportSummary, error) {
	i.summary.Start()

	outputBufWriter := bufio.NewWriter(i.output)

	articleDecoder := xml.NewDecoder(articleReader)
	priceDecoder := xml.NewDecoder(priceReader)

	var inElement string

XML_TOKEN:
	for {
		t, _ := articleDecoder.Token()
		if t == nil {
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			inElement = se.Name.Local
			if inElement == "item" {
				var a XmlArticle
				articleDecoder.DecodeElement(&a, &se)
				i.summary.Articles++
				categories := i.cfg.GetStringSlice("ignored.MAFT")
				for _, categorie := range categories {
					if a.Maft == categorie {
						i.summary.Ignored++
						continue XML_TOKEN
					}
				}
				categories = i.cfg.GetStringSlice("ignored.CAT1")
				for _, categorie := range categories {
					if a.Cat1 == categorie {
						i.summary.Ignored++
						continue XML_TOKEN
					}
				}
				p, err := i.getPrice(a.Id, priceDecoder)
				if err != nil {
					log.Println(err)
					continue XML_TOKEN
				}

				r := &Record{
					Id:             a.Id,
					IdPrefix:       i.cfg.GetString("id_prefix"),
					Description:    a.Description,
					PurchasePrice:  p.PurchasePrice,
					PurchaseFactor: i.cfg.GetFloat64("purchase_factor"),
					SellingFactor:  i.cfg.GetFloat64("selling_factor"),
					SellingPrice:   p.PurchasePrice * i.cfg.GetFloat64("selling_factor"),
					Category:       "Alltron",
					CategoryNumber: i.cfg.GetString("category_number"),
				}
				_, err = outputBufWriter.WriteString(r.FormatLine())
				if err != nil {
					return i.summary, err
				}
			}
		}

	}

	err := outputBufWriter.Flush()
	if err != nil {
		return i.summary, err
	}
	return i.summary, nil
}

func (i *AlltronImport) getPrice(id string, d *xml.Decoder) (*XmlArticlePrice, error) {
	var inElement string
	p, ok := i.prices[id]
	if ok {
		delete(i.prices, id)
		return &p, nil
	}
	for {
		t, _ := d.Token()
		if t == nil {
			return nil, fmt.Errorf("price not found for id: %s", id)
		}
		switch se := t.(type) {
		case xml.StartElement:
			inElement = se.Name.Local
			if inElement == "item" {
				var p XmlArticlePrice
				d.DecodeElement(&p, &se)
				if p.Id == id {
					return &p, nil
				} else {
					i.prices[p.Id] = p
				}
			}
		}

	}
}

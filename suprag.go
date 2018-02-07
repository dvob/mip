package mip

import (
	"archive/zip"
	"bufio"
	"bytes"
	"fmt"
	"github.com/spf13/viper"
	"github.com/tealeg/xlsx"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
)

type SupragImport struct {
	name        string
	cfg         *viper.Viper
	summary     *ImportSummary
	output      io.Writer
	initialized bool
}

func NewSupragImport(cfg *viper.Viper, output io.Writer) *SupragImport {
	return &SupragImport{
		name:    "Suprag",
		cfg:     cfg,
		summary: NewImportSummary(),
		output:  output,
	}
}

func (i *SupragImport) Name() string {
	return i.name
}

func (i *SupragImport) Init() error {
	i.initialized = true
	return nil
}

func (i *SupragImport) getXlsxFile() (*xlsx.File, error) {
	rawUrl := i.cfg.GetString("file")
	url, err := url.Parse(rawUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url:", rawUrl)
	}

	if url.Scheme == "" {
		log.Println("open local file")
		return xlsx.OpenFile(rawUrl)
	}

	if url.Scheme != "http" || url.Scheme == "https" {
		return nil, fmt.Errorf("scheme unsupported:", url.Scheme)
	}

	maxFileSize := i.cfg.GetInt64("max_download_file_size")

	var outputWriter io.Writer
	var content bytes.Buffer
	if i.cfg.GetBool("save_file") {
		filePath := filepath.Join(i.cfg.GetString("save_dir"), path.Base(rawUrl))
		saveFile, err := os.Create(filePath)
		if err != nil {
			return nil, err
		}
		outputWriter = io.MultiWriter(bufio.NewWriter(saveFile), &content)
	} else {
		outputWriter = &content
	}

	resp, err := http.Get(rawUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to download file: ", resp.Status)
	}

	if resp.ContentLength > maxFileSize {
		return nil, fmt.Errorf("file is to big")
	}

	log.Println("downloading file", rawUrl)
	_, err = io.Copy(outputWriter, io.LimitReader(resp.Body, maxFileSize))
	if err != nil {
		return nil, fmt.Errorf("failed to download data", err)
	}

	// check for unread bytes
	var p [1]byte
	n, err := resp.Body.Read(p[:])
	if n != 0 && err != io.EOF {
		return nil, fmt.Errorf("file is to big")
	}

	// content, err := ioutil.ReadAll(resp.Body)
	zipReader, err := zip.NewReader(bytes.NewReader(content.Bytes()), int64(content.Len()))
	if err != nil {
		return nil, err
	}
	return xlsx.ReadZipReader(zipReader)

}

func (i *SupragImport) Run() (*ImportSummary, error) {

	if !i.initialized {
		err := i.Init()
		if err != nil {
			return i.summary, err
		}
	}

	i.summary.Start()

	outputBufWriter := bufio.NewWriter(i.output)

	xlFile, err := i.getXlsxFile()
	if err != nil {
		return i.summary, fmt.Errorf("failed to open xlsx", err)
	}

	if len(xlFile.Sheets) < 1 {
		return i.summary, fmt.Errorf("no spreadsheetes in file")
	}

	sheet := xlFile.Sheets[0]

	lineNumber := i.cfg.GetInt("start_line") - 1
SUPRAG_XLSX:
	for _, row := range sheet.Rows[i.cfg.GetInt("start_line")-1:] {
		lineNumber++
		purchasePrice, err := row.Cells[7].Float()
		if err != nil {
			log.Println("failed to read line %d: could not parse %s", lineNumber, sheet.Rows[0].Cells[7])
			continue
		}
		sellingPrice, err := row.Cells[6].Float()
		if err != nil {
			log.Println("failed to read line %d: could not parse %s", lineNumber, sheet.Rows[0].Cells[6])
			continue
		}
		i.summary.Articles++
		//handle ignored manufacturers
		manufacturer := row.Cells[2].String()
		for _, ignored_manufacturer := range i.cfg.GetStringSlice("ignored_manufacturers") {
			if manufacturer == ignored_manufacturer {
				i.summary.Ignored++
				continue SUPRAG_XLSX
			}
		}
		r := &Record{
			Id:             row.Cells[0].String(),
			IdPrefix:       i.cfg.GetString("id_prefix"),
			Description:    row.Cells[9].String(),
			PurchasePrice:  purchasePrice,
			PurchaseFactor: i.cfg.GetFloat64("purchase_factor"),
			SellingFactor:  sellingPrice / purchasePrice,
			SellingPrice:   sellingPrice,
			Category:       "Suprag",
			CategoryNumber: i.cfg.GetString("category_number"),
		}
		_, err = outputBufWriter.WriteString(r.FormatLine())
		if err != nil {
			return i.summary, err
		}
	}
	outputBufWriter.Flush()
	return i.summary, nil
}

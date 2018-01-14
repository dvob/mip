package mip

import (
	"bufio"
	"fmt"
	"github.com/spf13/viper"
	"github.com/tealeg/xlsx"
	"io"
	"log"
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

func (i *SupragImport) Run() (*ImportSummary, error) {

	if !i.initialized {
		err := i.Init()
		if err != nil {
			return i.summary, err
		}
	}

	i.summary.Start()

	outputBufWriter := bufio.NewWriter(i.output)

	xlFile, err := xlsx.OpenFile(i.cfg.GetString("file"))
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

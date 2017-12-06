package mip

import (
	"bufio"
	"fmt"
	"github.com/spf13/viper"
	"github.com/tealeg/xlsx"
	"io"
	"log"
	"strings"
)

type MitelConfig struct {
	File                string             `json:"file"`
	SellingRepairFactor float64            `json:"selling_repair_factor"`
	PurchaseFactor      float64            `json:"purchase_factor"`
	IdPrefix            string             `json:"id_prefix"`
	Category            string             `json:"category"`
	CategoryNumber      string             `json:"category_number"`
	StartLine           int                `json:"start_line"`
	SellingFactors      map[string]float64 `json:"selling_factors"`
}

type MitelImport struct {
	name    string
	cfg     *viper.Viper
	summary *ImportSummary
	output  io.Writer
}

func NewMitelImport(cfg *viper.Viper, output io.Writer) *MitelImport {
	return &MitelImport{
		name:    "Mitel",
		cfg:     cfg,
		summary: NewImportSummary(),
		output:  output,
	}
}

func (i *MitelImport) Name() string {
	return i.name
}

func (i *MitelImport) getSellingFactor(name string) (float64, error) {
	value, ok := i.cfg.GetStringMap("selling_factors")[strings.ToLower(name)]
	if !ok {
		return 0.0, fmt.Errorf("can not get selling factor '%s'", name)
	}
	factor, ok := value.(int)
	if !ok {
		return 0.0, fmt.Errorf("selling factor '%s' is not an integer\n", name)
	}
	return float64(factor), nil
}

func (i *MitelImport) Run() (*ImportSummary, error) {

	i.summary.Start()

	// TODO: this should go into configuration
	IdColNr := 2                // CPQ
	SellingFactorNameColNr := 3 // MPG
	SellingPriceColNr := 4      // MLP
	RepairPriceColNr := 5       // Netto Rep Price
	DescriptionColNr := 8       // Bezeichnung

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
	for _, row := range sheet.Rows[i.cfg.GetInt("start_line")-1:] {
		lineNumber++
		if len(row.Cells)-1 < DescriptionColNr {
			log.Printf("skip line %d. only %d columns. line appears empty.\n", lineNumber, len(row.Cells))
			continue
		}
		sellingPrice, err := row.Cells[SellingPriceColNr].Float()
		if err != nil {
			log.Printf("failed to read line %d: could not parse selling price '%s'\n", lineNumber, row.Cells[SellingPriceColNr])
			continue
		}
		sellingFactorName := strings.Trim(row.Cells[SellingFactorNameColNr].String(), " ")
		sellingFactorPercent, err := i.getSellingFactor(sellingFactorName)
		if err != nil {
			log.Printf("could not get selling factor '%s': '%s'. skip row %d\n", sellingFactorName, err, lineNumber)
			continue
		}
		sellingFactor := 100.0 / (100.0 - sellingFactorPercent)
		r := &Record{
			Id:             row.Cells[IdColNr].String(),
			IdPrefix:       i.cfg.GetString("id_prefix"),
			Description:    row.Cells[DescriptionColNr].String(),
			PurchasePrice:  sellingPrice / sellingFactor,
			PurchaseFactor: i.cfg.GetFloat64("purchase_factor"),
			SellingFactor:  sellingFactor,
			SellingPrice:   sellingPrice,
			Category:       "Mitel",
			CategoryNumber: i.cfg.GetString("category_number"),
		}
		i.summary.Articles++
		_, err = outputBufWriter.WriteString(r.FormatLine())
		if err != nil {
			return i.summary, err
		}

		// check for repair price
		if row.Cells[RepairPriceColNr].String() == "" {
			continue
		}
		repairPrice, err := row.Cells[RepairPriceColNr].Float()
		if err != nil {
			log.Printf("could not parse repair price on row %d. skip repair\n", lineNumber)
			continue
		}
		r = &Record{
			Id:             row.Cells[IdColNr].String(),
			IdPrefix:       "REP-",
			Description:    "REPARATUR: " + row.Cells[DescriptionColNr].String(),
			PurchasePrice:  repairPrice,
			PurchaseFactor: i.cfg.GetFloat64("purchase_factor"),
			SellingFactor:  i.cfg.GetFloat64("selling_repair_factor"),
			SellingPrice:   repairPrice * i.cfg.GetFloat64("selling_repair_factor"),
			Category:       "Mitel",
			CategoryNumber: i.cfg.GetString("category_number"),
		}
		i.summary.Articles++
		_, err = outputBufWriter.WriteString(r.FormatLine())
		if err != nil {
			return i.summary, err
		}
	}
	err = outputBufWriter.Flush()
	if err != nil {
		return i.summary, err
	}

	return i.summary, nil
}

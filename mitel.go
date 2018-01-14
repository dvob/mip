package mip

import (
	"bufio"
	"fmt"
	"github.com/spf13/viper"
	"github.com/tealeg/xlsx"
	"io"
	"log"
	"regexp"
	"strings"
)

type MitelImport struct {
	name        string
	cfg         *viper.Viper
	summary     *ImportSummary
	output      io.Writer
	sheet       *xlsx.Sheet
	startLine   int
	column      *MitelColumns
	initialized bool
}

type MitelColumns struct {
	Id                *Column
	SellingFactorName *Column
	SellingPrice      *Column
	RepairPrice       *Column
	Description       *Column
}

type Column struct {
	Index int
	Regex *regexp.Regexp
}

func NewMitelImport(cfg *viper.Viper, output io.Writer) *MitelImport {
	return &MitelImport{
		name:    "Mitel",
		cfg:     cfg,
		summary: NewImportSummary(),
		output:  output,
		column: &MitelColumns{
			&Column{},
			&Column{},
			&Column{},
			&Column{},
			&Column{},
		},
	}
}

func (i *MitelImport) Name() string {
	return i.name
}

func (i *MitelImport) Init() error {
	xlFile, err := xlsx.OpenFile(i.cfg.GetString("file"))
	if err != nil {
		return fmt.Errorf("failed to open xlsx", err)
	}

	if len(xlFile.Sheets) < 1 {
		return fmt.Errorf("no spreadsheetes in file")
	}

	i.sheet = xlFile.Sheets[0]

	err = i.initColumns()
	if err != nil {
		return err
	}

	i.initialized = true
	return nil
}

func (i *MitelImport) initColumns() error {
	var err error
	i.column.Id.Regex, err = regexp.Compile(i.cfg.GetString("column_pattern.id"))
	i.column.SellingFactorName.Regex, err = regexp.Compile(i.cfg.GetString("column_pattern.selling_factor_name"))
	i.column.SellingPrice.Regex, err = regexp.Compile(i.cfg.GetString("column_pattern.selling_price"))
	i.column.RepairPrice.Regex, err = regexp.Compile(i.cfg.GetString("column_pattern.repair_price"))
	i.column.Description.Regex, err = regexp.Compile(i.cfg.GetString("column_pattern.description"))

	if err != nil {
		return err
	}

	found := false
	for index, row := range i.sheet.Rows {
		found = findColumns(i, row)
		if found {
			i.startLine = index + 1
			break
		}
	}
	if !found {
		return fmt.Errorf("could not find header line")
	}
	return nil
}

func findColumns(i *MitelImport, row *xlsx.Row) bool {
COLUMN:
	for _, column := range []*Column{
		i.column.Id,
		i.column.SellingFactorName,
		i.column.SellingPrice,
		i.column.RepairPrice,
		i.column.Description} {

		for index, cell := range row.Cells {
			if column.Regex.FindStringIndex(cell.String()) != nil {
				column.Index = index
				continue COLUMN
			}
		}
		// fmt.Println("could not find: ", column.Regex.String())
		return false
	}
	return true
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

	if !i.initialized {
		err := i.Init()
		if err != nil {
			return i.summary, err
		}
	}

	i.summary.Start()

	outputBufWriter := bufio.NewWriter(i.output)

	lineNumber := i.startLine + 1
	for _, row := range i.sheet.Rows[i.startLine:] {
		lineNumber++
		if len(row.Cells)-1 < i.column.Description.Index {
			log.Printf("skip line %d. only %d columns. line appears empty.\n", lineNumber, len(row.Cells))
			continue
		}
		sellingPrice, err := row.Cells[i.column.SellingPrice.Index].Float()
		if err != nil {
			log.Printf("failed to read line %d: could not parse selling price '%s'\n", lineNumber, row.Cells[i.column.SellingPrice.Index])
			continue
		}
		sellingFactorName := strings.Trim(row.Cells[i.column.SellingFactorName.Index].String(), " ")
		sellingFactorPercent, err := i.getSellingFactor(sellingFactorName)
		if err != nil {
			log.Printf("could not get selling factor '%s': '%s'. skip row %d\n", sellingFactorName, err, lineNumber)
			continue
		}
		sellingFactor := 100.0 / (100.0 - sellingFactorPercent)
		r := &Record{
			Id:             row.Cells[i.column.Id.Index].String(),
			IdPrefix:       i.cfg.GetString("id_prefix"),
			Description:    row.Cells[i.column.Description.Index].String(),
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
		if row.Cells[i.column.RepairPrice.Index].String() == "" {
			continue
		}
		repairPrice, err := row.Cells[i.column.RepairPrice.Index].Float()
		if err != nil {
			log.Printf("could not parse repair price on row %d. skip repair\n", lineNumber)
			continue
		}
		r = &Record{
			Id:             row.Cells[i.column.Id.Index].String(),
			IdPrefix:       "REP-",
			Description:    "REPARATUR: " + row.Cells[i.column.Description.Index].String(),
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
	err := outputBufWriter.Flush()
	if err != nil {
		return i.summary, err
	}

	return i.summary, nil
}

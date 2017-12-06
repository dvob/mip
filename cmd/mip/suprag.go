package main

import (
	"fmt"
	"github.com/dsbrng25b/mip"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

var supragCmd = &cobra.Command{
	Use:   "suprag [xlsx_file]",
	Short: "perform the suprag import",
	Args:  ZeroOrNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		file, err := os.Create(viper.GetString("output_file"))
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to open output file: ", err)
			os.Exit(1)
		}
		defer file.Close()
		export, err := mip.NewExport(file, viper.GetString("output_encoding"))
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to initialize export: ", err)
			os.Exit(1)
		}

		if len(args) > 0 {
			viper.Set("suprag.file", args[0])
		}

		imp := mip.NewSupragImport(viper.Sub("suprag"), export)
		runImport(imp)
	},
}

func init() {

	RootCmd.AddCommand(supragCmd)

}

package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"mip"
	"os"
)

var mitelCmd = &cobra.Command{
	Use:   "mitel [xlsx_file]",
	Short: "perform the mitel import",
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
			viper.Set("mitel.file", args[0])
		}

		imp := mip.NewMitelImport(viper.Sub("mitel"), export)
		runImport(imp)
	},
}

func init() {

	RootCmd.AddCommand(mitelCmd)

}

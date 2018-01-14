package main

import (
	"fmt"
	"github.com/dsbrng25b/mip"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"os"
)

var allCmd = &cobra.Command{
	Use:   "all",
	Short: "perform all imports (alltron, suprag, mitel)",
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

		imports := []mip.Importer{
			mip.NewAlltronImport(viper.Sub("alltron"), export),
			mip.NewMitelImport(viper.Sub("mitel"), export),
			mip.NewSupragImport(viper.Sub("suprag"), export)}

		// initialize importer
		for _, imp := range imports {
			err := imp.Init()
			if err != nil {
				fmt.Fprintln(os.Stderr, "faild to initialize", imp.Name(), ": ", err)
				os.Exit(1)
			}
		}

		// start processing
		all_ps := mip.StartImportSummary()
		log.Println("ALL:", "start processing")
		for _, imp := range imports {
			is, err := runImport(imp)
			if err != nil {
				os.Exit(1)
			}
			all_ps.Add(is)
		}
		all_ps.Stop()
		log.Println("ALL:", all_ps)

	},
}

func init() {

	RootCmd.AddCommand(allCmd)

}

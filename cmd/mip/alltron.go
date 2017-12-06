package main

import (
	"fmt"
	"github.com/dsbrng25b/mip"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

var alltronCmd = &cobra.Command{
	Use:   "alltron [article_file price_file]",
	Short: "perform the alltron import",
	Args:  ZeroOrNArgs(2),
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

		// if path to files are passed by arguments set for ftp and local
		if len(args) >= 2 {
			viper.Set("alltron.article_file", args[0])
			viper.Set("alltron.ftp_article_file", args[0])
			viper.Set("alltron.price_file", args[1])
			viper.Set("alltron.ftp_price_file", args[1])
		}

		imp := mip.NewAlltronImport(viper.Sub("alltron"), export)
		runImport(imp)
	},
}

func init() {

	viper.SetDefault("alltron.use_ftp", true)
	viper.SetDefault("alltron.ftp_save_files", false)
	alltronCmd.Flags().Bool("ftp", true, "process files from ftp server")
	alltronCmd.Flags().Bool("save", false, "when downloading files from ftp additionally save them locally")
	viper.BindPFlag("alltron.use_ftp", alltronCmd.Flags().Lookup("ftp"))
	viper.BindPFlag("alltron.ftp_save_files", alltronCmd.Flags().Lookup("save"))
	RootCmd.AddCommand(alltronCmd)

}

package main

import (
	"fmt"
	"github.com/dsbrng25b/mip"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"os"
)

var (
	cfgFile    string
	dumpFormat string

	// set during build
	version string
	commit  string
	date    string
)

var RootCmd = &cobra.Command{
	Use:   "mip",
	Short: "messerli import preparer",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "print the version number of mip",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("version: ", version)
		fmt.Println("git commit: ", commit)
		fmt.Println("build date: ", date)
	},
}

var listEncCmd = &cobra.Command{
	Use:   "list-enc",
	Short: "list available output encodings",
	Run: func(cmd *cobra.Command, args []string) {
		for enc, _ := range mip.Encodings {
			fmt.Println(enc)
		}
	},
}

func init() {

	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "config.yaml", "configuration file")
	RootCmd.PersistentFlags().StringP("output", "o", "output.csv", "output file")

	viper.BindPFlag("output", RootCmd.PersistentFlags().Lookup("output"))

	RootCmd.AddCommand(versionCmd)
	RootCmd.AddCommand(listEncCmd)
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Can't read config:", err)
		os.Exit(1)
	}
}

func runImport(i mip.Importer) (*mip.ImportSummary, error) {
	log.Println(i.Name(), "start processing")
	is, err := i.Run()
	log.Println(i.Name(), is)
	if err != nil {
		log.Fatal(i.Name(), "failed", err)
	}
	log.Println(i.Name(), "processing finished")
	return is, nil
}

func ZeroOrNArgs(i int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == i || len(args) == 0 {
			return nil
		}
		return fmt.Errorf("accepts either exact %d or no arguments", i)
	}
}

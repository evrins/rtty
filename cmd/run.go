package cmd

import (
	"fmt"
	"github.com/skanehira/rtty/service"
	"github.com/skanehira/rtty/utils"
	"github.com/spf13/cobra"
)

var command = utils.GetEnv("SHELL", "bash")

var runCmd = &cobra.Command{
	Use:   "run [Command]",
	Short: fmt.Sprintf("Run specified command (default \"%s\")", command),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		if len(args) > 0 {
			command = args[0]
		}
		port, err := cmd.PersistentFlags().GetInt("port")
		if err != nil {
			return
		}

		font, err := cmd.PersistentFlags().GetString("font")
		if err != nil {
			return
		}
		fontSize, err := cmd.PersistentFlags().GetString("font-size")
		if err != nil {
			return
		}
		addr, err := cmd.PersistentFlags().GetString("addr")
		if err != nil {
			return
		}
		openView, err := cmd.PersistentFlags().GetBool("view")
		if err != nil {
			return
		}
		consul, err := cmd.PersistentFlags().GetString("consul")
		if err != nil {
			return
		}
		err = service.StartWebService(addr, port, font, fontSize, consul, openView)
		if err != nil {
			return
		}
		return
	},
}

func init() {
	runCmd.PersistentFlags().IntP("port", "p", 9999, "server port")
	runCmd.PersistentFlags().StringP("addr", "a", "0.0.0.0", "server address")
	runCmd.PersistentFlags().String("font", "", "font")
	runCmd.PersistentFlags().String("font-size", "", "font size")
	runCmd.PersistentFlags().BoolP("view", "v", false, "open browser")
	runCmd.PersistentFlags().String("consul", "192.168.112.26:8500", "consul address")
	rootCmd.AddCommand(runCmd)
}

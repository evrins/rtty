package cmd

import (
	"fmt"
	"github.com/skanehira/rtty/service"
	"github.com/skanehira/rtty/utils"
	"github.com/spf13/cobra"
	"strconv"
)

var command = utils.GetEnv("SHELL", "bash")

var runCmd = &cobra.Command{
	Use:   "run [Command]",
	Short: fmt.Sprintf("Run specified command (default \"%s\")", command),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		if len(args) > 0 {
			command = args[0]
		}
		portFlag, err := cmd.PersistentFlags().GetInt("port")
		if err != nil {
			return
		}
		port := strconv.Itoa(portFlag)

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

		err = service.StartWebService(addr, port, font, fontSize, openView)
		if err != nil {
			return
		}
		return
	},
}

func init() {
	runCmd.PersistentFlags().IntP("port", "p", 9999, "server port")
	runCmd.PersistentFlags().StringP("addr", "a", "localhost", "server address")
	runCmd.PersistentFlags().String("font", "", "font")
	runCmd.PersistentFlags().String("font-size", "", "font size")
	runCmd.PersistentFlags().BoolP("view", "v", false, "open browser")
	rootCmd.AddCommand(runCmd)
}

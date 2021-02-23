package pages

import (
	"github.com/cli/cli/pkg/cmdutil"
	"github.com/spf13/cobra"
	pagesDeployCmd "github.com/cli/cli/pkg/cmd/pages/deploy"
)

func NewCmdPages(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pages <command>",
		Short: "Login, logout, and refresh your authentication",
		Long:  `Manage gh's authentication state.`,
	}

	cmdutil.DisableAuthCheck(cmd)

	cmd.AddCommand(pagesDeployCmd.NewCmdDeploy(f, nil))

	return cmd
}

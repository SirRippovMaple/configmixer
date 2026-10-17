package cmd

import "github.com/spf13/cobra"

// NewRootCmd builds the configmixer root command.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "configmixer",
		Short:         "Mix disjoint config mixins into a single .env file",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newGenerateCmd())
	root.AddCommand(newListCmd())
	return root
}

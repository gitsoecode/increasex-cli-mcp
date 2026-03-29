package cli

import (
	"github.com/jessevaughan/increasex/internal/mcp"
	"github.com/spf13/cobra"
)

func newMCPCmd(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{Use: "mcp", Short: "Run the local MCP server"}
	cmd.AddCommand(&cobra.Command{
		Use:   "serve",
		Short: "Serve MCP over stdio",
		RunE: func(cmd *cobra.Command, args []string) error {
			server := mcp.NewServer(ctx.Services, mcp.Options{
				Profile:     ctx.Options.Profile,
				Environment: ctx.Options.Environment,
				APIKey:      ctx.Options.APIKey,
				Debug:       ctx.Options.Debug,
			})
			return server.Serve(cmd.Context())
		},
	})
	return cmd
}

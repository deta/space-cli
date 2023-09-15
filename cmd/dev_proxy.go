package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/deta/space/cmd/utils"
	"github.com/deta/space/internal/proxy"
	"github.com/deta/space/internal/runtime"
	"github.com/deta/space/internal/spacefile"
	"github.com/deta/space/pkg/components/emoji"
	"github.com/deta/space/pkg/components/styles"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

func newCmdDevProxy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proxy",
		Short: "Start a reverse proxy for your micros",
		Long: `Start a reverse proxy for your micros

The micros will be automatically discovered and proxied to.`,
		PreRunE:  utils.CheckProjectInitialized("dir"),
		PostRunE: utils.CheckLatestVersion,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			directory, _ := cmd.Flags().GetString("dir")
			host, _ := cmd.Flags().GetString("host")
			port, _ := cmd.Flags().GetInt("port")
			open, _ := cmd.Flags().GetBool("open")

			if !cmd.Flags().Changed("port") {
				port, err = GetFreePort(utils.DevPort)
				if err != nil {
					return fmt.Errorf("failed to get free port: %w", err)
				}
			}

			if err := devProxy(directory, host, port, open); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringP("dir", "d", ".", "directory of the project")
	cmd.Flags().IntP("port", "p", 0, "port to run the proxy on")
	cmd.Flags().StringP("host", "H", "localhost", "host to run the proxy on")
	cmd.Flags().Bool("open", false, "open the app in the browser")

	return cmd
}

func devProxy(projectDir string, host string, port int, open bool) error {
	meta, err := runtime.GetProjectMeta(projectDir)
	if err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%d", host, port)

	microDir := filepath.Join(projectDir, ".space", "micros")
	spacefile, _ := spacefile.LoadSpacefile(projectDir)

	if entries, err := os.ReadDir(microDir); err != nil || len(entries) == 0 {
		utils.Logger.Printf("%s No running micros detected.", emoji.X)
		utils.Logger.Printf("L Use %s to manually start a micro", styles.Blue("space dev up <micro>"))
		return err
	}

	projectKey, err := utils.GenerateDataKeyIfNotExists(meta.ID)
	if err != nil {
		return fmt.Errorf("failed to generate project key: %w", err)
	}

	reverseProxy := proxy.NewReverseProxy(projectKey, meta.ID, meta.Name, meta.Alias)
	if err := loadMicrosFromDir(reverseProxy, spacefile.Micros, microDir); err != nil {
		return err
	}
	if err != nil {
		return err
	}
	server := &http.Server{
		Addr:    addr,
		Handler: reverseProxy,
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		utils.Logger.Printf("%s proxy listening on http://%s", emoji.Laptop, addr)
		server.ListenAndServe()
	}()

	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
		utils.Logger.Printf("\n\nShutting down...\n\n")
		server.Shutdown(context.Background())
	}()

	if open {
		browser.OpenURL(fmt.Sprintf("http://localhost:%d", port))
	}

	wg.Wait()
	return nil
}

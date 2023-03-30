package dev

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/deta/space/cmd/shared"
	"github.com/deta/space/internal/spacefile"
	"github.com/deta/space/pkg/components/emoji"
	"github.com/deta/space/pkg/components/styles"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

func newCmdDevProxy() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "proxy",
		Short:   "Start the proxy server for your running micros",
		PreRunE: shared.CheckProjectInitialized("dir"),
		Run: func(cmd *cobra.Command, args []string) {
			var err error

			directory, _ := cmd.Flags().GetString("dir")
			host, _ := cmd.Flags().GetString("host")
			port, _ := cmd.Flags().GetInt("port")
			open, _ := cmd.Flags().GetBool("open")

			if !cmd.Flags().Changed("port") {
				port, err = GetFreePort(devDefaultPort)
				if err != nil {
					shared.Logger.Printf("%s Failed to get free port: %s", emoji.ErrorExclamation, err)
					os.Exit(1)
				}
			}

			if err := devProxy(directory, host, port, open); err != nil {
				os.Exit(1)
			}

		},
	}

	cmd.Flags().StringP("dir", "d", ".", "directory of the project")
	cmd.Flags().IntP("port", "p", devDefaultPort, "port to run the proxy on")
	cmd.Flags().StringP("host", "H", "localhost", "host to run the proxy on")
	cmd.Flags().Bool("open", false, "open the app in the browser")

	return cmd
}

func devProxy(projectDir string, host string, port int, open bool) error {

	addr := fmt.Sprintf("%s:%d", host, port)

	microDir := filepath.Join(projectDir, ".space", "micros")
	spacefile, _ := spacefile.ParseSpacefile(projectDir)

	if entries, err := os.ReadDir(microDir); err != nil || len(entries) == 0 {
		shared.Logger.Printf("%s No running micros detected.", emoji.X)
		shared.Logger.Printf("L Use %s to manually start a micro", styles.Blue("space dev up <micro>"))
		os.Exit(1)
	}

	reverseProxy, err := proxyFromDir(spacefile.Micros, microDir)
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
		shared.Logger.Printf("%s proxy listening on http://%s", emoji.Laptop, addr)
		server.ListenAndServe()
	}()

	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
		shared.Logger.Printf("\n\nShutting down...\n\n")
		server.Shutdown(context.Background())
	}()

	if open {
		browser.OpenURL(fmt.Sprintf("http://localhost:%d", port))
	}

	wg.Wait()
	return nil
}

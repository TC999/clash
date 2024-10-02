package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/doreamon-design/clash"
	"github.com/doreamon-design/clash/config"
	"github.com/doreamon-design/clash/hub"
	"github.com/doreamon-design/clash/hub/executor"
	"github.com/doreamon-design/clash/log"
	"github.com/go-zoox/cli"
	"go.uber.org/automaxprocs/maxprocs"

	C "github.com/doreamon-design/clash/constant"
)

func main() {
	app := cli.NewSingleProgram(&cli.SingleProgramConfig{
		Name:    "clash",
		Usage:   "A rule-based tunnel in Go",
		Version: clash.Version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "home-dir",
				Usage:   "set configuration directory",
				EnvVars: []string{"CLASH_HOME_DIR"},
				Aliases: []string{"d"},
			},
			&cli.StringFlag{
				Name:    "config",
				Usage:   "specify configuration file",
				EnvVars: []string{"CLASH_CONFIG_FILE"},
				Aliases: []string{"f"},
			},
			&cli.StringFlag{
				Name:    "ext-ui",
				Usage:   "override external ui directory",
				EnvVars: []string{"CLASH_OVERRIDE_EXTERNAL_UI_DIR"},
			},
			&cli.StringFlag{
				Name:    "ext-ctl",
				Usage:   "override external controller address",
				EnvVars: []string{"CLASH_OVERRIDE_EXTERNAL_CONTROLLER"},
			},
			&cli.StringFlag{
				Name:    "secret",
				Usage:   "override secret for RESTful API",
				EnvVars: []string{"CLASH_OVERRIDE_SECRET"},
			},
			&cli.BoolFlag{
				Name:    "test",
				Usage:   "test configuration and exit",
				Aliases: []string{"t"},
			},
		},
	})

	app.Command(func(ctx *cli.Context) error {
		homeDir := ctx.String("home-dir")
		configFile := ctx.String("config")
		externalUI := ctx.String("ext-ui")
		externalController := ctx.String("ext-ctl")
		secret := ctx.String("secret")
		testConfig := ctx.Bool("test")

		maxprocs.Set(maxprocs.Logger(func(string, ...any) {}))

		if homeDir != "" {
			if !filepath.IsAbs(homeDir) {
				currentDir, _ := os.Getwd()
				homeDir = filepath.Join(currentDir, homeDir)
			}
			C.SetHomeDir(homeDir)
		}

		if configFile != "" {
			if !filepath.IsAbs(configFile) {
				currentDir, _ := os.Getwd()
				configFile = filepath.Join(currentDir, configFile)
			}
			C.SetConfig(configFile)
		} else {
			configFile := filepath.Join(C.Path.HomeDir(), C.Path.Config())
			C.SetConfig(configFile)
		}

		if err := config.Init(C.Path.HomeDir()); err != nil {
			return fmt.Errorf("failed to initial configuration directory: %s", err.Error())
		}

		if testConfig {
			if _, err := executor.Parse(); err != nil {
				log.Errorln(err.Error())
				return fmt.Errorf("configuration file %s test failed", C.Path.Config())
			}

			fmt.Printf("configuration file %s test is successful\n", C.Path.Config())
			return nil
		}

		var options []hub.Option
		if externalUI != "" {
			options = append(options, hub.WithExternalUI(externalUI))
		}
		if externalController != "" {
			options = append(options, hub.WithExternalController(externalController))
		}
		if secret != "" {
			options = append(options, hub.WithSecret(secret))
		}

		if err := hub.Parse(options...); err != nil {
			return fmt.Errorf("failed to parse config: %s", err.Error())
		}

		termSign := make(chan os.Signal, 1)
		hupSign := make(chan os.Signal, 1)
		signal.Notify(termSign, syscall.SIGINT, syscall.SIGTERM)
		signal.Notify(hupSign, syscall.SIGHUP)
		for {
			select {
			case <-termSign:
				return nil
			case <-hupSign:
				if cfg, err := executor.ParseWithPath(C.Path.Config()); err == nil {
					executor.ApplyConfig(cfg, true)
				} else {
					log.Errorln("failed to parse config: %s", err.Error())
				}
			}
		}
	})

	app.Run()
}

// T2C Phantom — application de bureau (macOS, Windows, Linux).
//
// Une vraie fenêtre native : le tableau de bord n'ouvre plus le navigateur.
// L'interface (pkg/api/web) et ses routes /api sont servies directement dans
// la fenêtre, sans port réseau ouvert ; seul le proxy .t2c écoute sur
// 127.0.0.1:8080 pour la navigation dans le navigateur.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"t2c/pkg/api"
	"t2c/pkg/buildinfo"
	"t2c/pkg/utils"
)

// version : fixée au build (-ldflags "-X main.version=v2.0.0").
var version = "dev"

func main() {
	buildinfo.Version = version // annoncée aux relais (avis de version signé)
	_ = os.MkdirAll(utils.GetT2CDataDir(), 0o700)
	utils.InitSecurityLogger(utils.GetPath("security.log"))

	d := api.NewDashboard()
	err := wails.Run(&options.App{
		Title:     "T2C Phantom",
		Width:     1200,
		Height:    820,
		MinWidth:  900,
		MinHeight: 620,
		AssetServer: &assetserver.Options{
			Assets:  api.Assets(),
			Handler: d.APIHandler(), // /api/* (tout ce qui n'est pas un fichier de l'interface)
		},
		BackgroundColour: &options.RGBA{R: 10, G: 10, B: 10, A: 255},
		OnShutdown:       func(context.Context) { d.Shutdown() },
		// premier lancement : proposer l'installation (Applications / Bureau / menu)
		OnDomReady: func(ctx context.Context) { go offerInstall(ctx) },
		Mac: &mac.Options{
			About: &mac.AboutInfo{
				Title:   "T2C Phantom " + version,
				Message: "Réseau de pairs pseudonymes — TUTODECODE",
			},
		},
		Windows: &windows.Options{},
		Linux:   &linux.Options{ProgramName: "t2c-phantom"},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "T2C Phantom:", err)
		os.Exit(1)
	}
}

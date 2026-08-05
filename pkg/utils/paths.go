package utils

import (
	"os"
	"path/filepath"
	"runtime"
)

// GetT2CDataDir renvoie le dossier de données et de cache approprié en fonction de l'OS et de l'utilisateur
func GetT2CDataDir() string {
	// 1. Si une variable d'environnement personnalisée est spécifiée, elle est prioritaire
	if envDir := os.Getenv("T2C_DATA_DIR"); envDir != "" {
		return envDir
	}

	// 2. Détection de l'utilisateur root/nobody (Services système)
	isSystemUser := false
	if runtime.GOOS != "windows" {
		uid := os.Getuid()
		// uid 0 (root), 99 (nobody sur Centos/RHEL/macOS), 65534 (nobody sur Debian/Ubuntu)
		if uid == 0 || uid == 99 || uid == 65534 {
			isSystemUser = true
		}
	}

	if isSystemUser {
		switch runtime.GOOS {
		case "linux":
			return "/var/lib/t2c-phantom"
		case "darwin":
			return "/var/db/t2c-phantom"
		}
	}

	// 3. Mode utilisateur standard (Caches de l'utilisateur courant)
	userDir, err := os.UserCacheDir()
	if err == nil {
		return filepath.Join(userDir, "t2c-phantom")
	}

	// 4. Fallback de dernier recours (Dossier courant)
	return "data"
}

// GetPath retourne le chemin complet pour un sous-élément de données
func GetPath(subPath string) string {
	return filepath.Join(GetT2CDataDir(), subPath)
}

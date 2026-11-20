// Package buildinfo : version de l'application ou du relais, fixée au build
// des releases (-ldflags "-X t2c/pkg/buildinfo.Version=v2.6.0"). « dev » hors
// release : jamais de bannière de mise à jour, refusée par les relais dès que
// l'avis de version fixe un minimum (voir trust.VersionNotice).
package buildinfo

var Version = "dev"

package proxy

import (
	"fmt"
	"net/http"
)

// WrongCertMessage : la CA locale n'est pas celle générée par T2C Phantom.
const WrongCertMessage = "Accès refusé : CA locale « tutodecode » absente ou invalide — relance T2C Phantom pour la régénérer."

// refuseWrongCert bloque le HTTPS si la CA locale n'est pas utilisable.
func (s *Server) refuseWrongCert(w http.ResponseWriter) bool {
	if s.CA == nil || !s.CA.IsTutodecodeCA() {
		http.Error(w, WrongCertMessage, http.StatusForbidden)
		fmt.Println("🚫", WrongCertMessage)
		return true
	}
	return false
}

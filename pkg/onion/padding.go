package onion

import (
	"crypto/rand"
	"encoding/binary"
)

// Remplissage : chaque commande est enveloppée et complétée jusqu'à l'une de
// quelques tailles fixes. Un observateur (ou un relais intermédiaire) ne voit
// plus la taille réelle d'une demande ou d'une réponse : une recherche, une
// demande de catalogue et une cellule leurre sont indiscernables.
//
// Le remplissage est aléatoire (et non des zéros) : même chiffré, rien ne
// distingue une cellule pleine d'une cellule remplie.

// MaxCell : taille maximale d'une commande enveloppée (doit tenir, avec trois
// couches de chiffrement, dans une trame wire).
const MaxCell = 266_176

// cellSizes : tailles possibles, de la plus petite à la plus grande.
var cellSizes = []int{1024, 8192, 66_560, MaxCell}

// Pad enveloppe p (une commande complète) à la plus petite taille fixe qui la contient.
func Pad(p []byte) ([]byte, bool) {
	need := 5 + len(p)
	for _, size := range cellSizes {
		if need <= size {
			out := make([]byte, size)
			out[0] = CmdPadded
			binary.BigEndian.PutUint32(out[1:5], uint32(len(p)))
			copy(out[5:], p)
			_, _ = rand.Read(out[need:])
			return out, true
		}
	}
	return nil, false
}

// Unpad retire l'enveloppe (ok=false si b n'en est pas une valide).
func Unpad(b []byte) ([]byte, bool) {
	if len(b) < 5 || b[0] != CmdPadded {
		return nil, false
	}
	n := int(binary.BigEndian.Uint32(b[1:5]))
	if n == 0 || 5+n > len(b) {
		return nil, false
	}
	return b[5 : 5+n], true
}

// Drop : cellule leurre de la plus petite taille.
func Drop() []byte {
	b, _ := Pad([]byte{CmdDrop})
	return b
}

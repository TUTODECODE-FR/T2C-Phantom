package sysint

import "testing"

func TestOpenURLRejects(t *testing.T) {
	for _, u := range []string{"file:///etc/passwd", "javascript:alert(1)", "http://a\nb", "-a Calculator"} {
		if OpenURL("nope", u) == nil {
			t.Fatalf("%q accepté", u)
		}
	}
	if err := OpenURL("navigateur-inexistant", "http://tutodecode.t2c/"); err == nil || err.Error() != "navigateur introuvable" {
		t.Fatalf("adresse valide refusée: %v", err)
	}
}

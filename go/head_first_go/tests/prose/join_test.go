package prose

import "testing"

func TestTwoElements(t *testing.T) {
	list := []string{"mela", "arancia"}
	if JoinWithCommas(list) != "mela e arancia" {
		t.Error("valore aspettato non matchato")

	}
}
func TestThreeElements(t *testing.T) {
	list := []string{"mela", "arancia", "pera"}
	if JoinWithCommas(list) != "mela, arancia e pera" {
		t.Error("valore aspettato non matchato")

	}
}

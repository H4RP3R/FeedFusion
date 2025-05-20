package censor

import (
	"path/filepath"
	"testing"
)

func TestCensor_Check(t *testing.T) {
	var c Censor

	jsonPath := filepath.Join("test_data", "words.json")
	if err := c.LoadFromJSON(jsonPath); err != nil {
		t.Fatalf("failed to load words: %v", err)
	}

	tests := []struct {
		name    string
		comment string
		want    bool
	}{
		{"No match", "hello world", false},

		{"Match single word", "скрепец", true},
		{"Exception word", "скрепа", false},
		{"Banned derivative", "скрепаносец", true},
		{"Mixed text", "скрепа и скрепец", true},
		{"Exception plural", "скрепки", false},

		{"Match base word", "ты полный хуй", true},
		{"Match derivative хуёвый", "хуёвый день", true},
		{"Match derivative хуита", "эта хуита меня достала", true},
		{"Match derivative хуйня", "по телеку одна хуйня", true},
		{"Match derivative нихуя", "мне нихуя не заплатили", true},
		{"Match derivative охуел", "ты совсем охуел", true},
		{"Match derivative подохуевший", "в конец подохуевший", true},
		{"Match derivative хуячу", "я хуячу без выходных", true},
		{"Match derivative хуя", "хуя ты модный", true},
		{"Match derivative нахуячился", "Вася нахуячился", true},
		{"Match derivative хуем", "бить хуем по балалайке", true},
		{"Exception подстрахуй", "подстрахуй меня, пожалуйста", false},
		{"False positive result to avoid 1", "съел вкусную хурму", false},
		{"False positive result to avoid 2", "пахучий цветок", false},
		{"False positive result to avoid 3", "пахучий и хурма", false},
		{"Mixed text 1", "подстрахуй, ты хуй", true},
		{"Mixed text 2", "эй ты хуй, подстрахуй", true},
		{"Homograph attack 1 (ASCII x)", "xуй", true},
		{"Homograph attack 2 (ASCII o,x,y)", "пoxyй", true},
		{"Homograph attack 3 (ASCII e)", "охуeл", true},
		{"Homograph attack 4 (Greek letters)", "ОⲭⲩⲉЛ", true},
		{"Mixed script attack", "xүй", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.Check(tt.comment)
			if got != tt.want {
				t.Errorf("Check(%q) = %v; want %v", tt.comment, got, tt.want)
			}
		})
	}
}

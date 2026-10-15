package territory

import "testing"

func BenchmarkBuildSnapshot(b *testing.B) {
	for b.Loop() {
		if _, err := buildEmbedded(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkVillage(b *testing.B) {
	if err := Load(); err != nil {
		b.Fatal(err)
	}
	for b.Loop() {
		if _, err := VillageByCode("31.71.01.1001"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkVillagesOf(b *testing.B) {
	if err := Load(); err != nil {
		b.Fatal(err)
	}
	for b.Loop() {
		if _, err := VillagesOf("31.71.01"); err != nil {
			b.Fatal(err)
		}
	}
}

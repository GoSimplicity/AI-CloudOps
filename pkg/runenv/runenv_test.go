package runenv

import "testing"

func TestIsProductionValue(t *testing.T) {
	for _, v := range []string{"production", " Production ", "PRODUCTION", "\tproduction\n"} {
		if !IsProductionValue(v) {
			t.Fatalf("%q 应判定为生产环境", v)
		}
	}
	for _, v := range []string{"", "development", "test", "prod", "production1", "pro duction"} {
		if IsProductionValue(v) {
			t.Fatalf("%q 不应判定为生产环境", v)
		}
	}
}

func TestIsProductionReadsEnv(t *testing.T) {
	t.Setenv(EnvKey, " PRODUCTION ")
	if !IsProduction() {
		t.Fatal("ENV=' PRODUCTION ' 应判定为生产环境")
	}
	t.Setenv(EnvKey, "development")
	if IsProduction() {
		t.Fatal("ENV=development 不应判定为生产环境")
	}
}

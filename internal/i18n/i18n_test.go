package i18n

import (
	"reflect"
	"testing"
)

func TestCatalogCompleteness(t *testing.T) {
	zh := For(LangZH)
	en := For(LangEN)

	if zh.LangCode != LangZH || en.LangCode != LangEN {
		t.Fatalf("unexpected catalog lang codes: zh=%q en=%q", zh.LangCode, en.LangCode)
	}

	vZH := reflect.ValueOf(zh)
	vEN := reflect.ValueOf(en)
	typ := vZH.Type()
	for i := 0; i < typ.NumField(); i++ {
		name := typ.Field(i).Name
		if vZH.Field(i).String() == "" {
			t.Errorf("zhCatalog.%s is empty", name)
		}
		if vEN.Field(i).String() == "" {
			t.Errorf("enCatalog.%s is empty", name)
		}
	}

	if Normalize("en-US") != LangEN || Normalize("zh_CN.UTF-8") != LangZH || Normalize("") != LangZH {
		t.Fatal("Normalize behaved unexpectedly")
	}
}

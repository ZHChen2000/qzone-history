package export

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"qzone-history/internal/domain/entity"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestWriteViewerHTMLPreservesPayload(t *testing.T) {
	payload := ViewerPayload{UserQQ: "test", GeneratedAt: "2026-10-06 02:29:07", Moments: []entity.Moment{{ID: "one", UserQQ: "test", Content: "<script>alert('test')</script>", TimeText: "\\t2020年9月30日 23:59\\t", IsReconstructed: true}}}
	filename := filepath.Join(t.TempDir(), "viewer.html")
	if err := WriteViewerHTML(filename, payload); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	html := string(raw)
	match := regexp.MustCompile("const DATA = JSON.parse\\(decodePayload\\(\"([^\"]+)\"\\)\\);").FindStringSubmatch(html)
	if len(match) != 2 {
		t.Fatal("embedded payload missing")
	}
	decoded, err := base64.StdEncoding.DecodeString(match[1])
	if err != nil {
		t.Fatal(err)
	}
	var got ViewerPayload
	if err := json.Unmarshal(decoded, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, payload) {
		t.Fatal("viewer changed source payload")
	}
	if strings.Contains(html, "{{.PayloadB64}}") || strings.Contains(html, payload.Moments[0].Content) {
		t.Fatal("unsafe or unresolved payload")
	}
	for _, marker := range []string{"最新在前", "最早在前", "折叠疑似重复", "逐条显示", "function groupMoments("} {
		if !strings.Contains(html, marker) {
			t.Errorf("missing feature: %s", marker)
		}
	}
}

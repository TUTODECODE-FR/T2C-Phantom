package proxy

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"t2c/pkg/trust"
)

type fakeSites struct{ cat *trust.Catalog }

func (f fakeSites) SiteDir(context.Context, string) (string, *trust.SiteEntry, error) {
	return "", nil, context.Canceled
}
func (f fakeSites) Resolve(context.Context, string) (*trust.SiteEntry, error) { return nil, nil }
func (f fakeSites) Catalog(context.Context) (*trust.Catalog, error)           { return f.cat, nil }

func TestSitesList(t *testing.T) {
	s := &Server{Node: fakeSites{&trust.Catalog{Version: 7, Sites: []trust.SiteEntry{
		{ID: "repo", Alias: "repo", Title: "Dépôts", Size: 10},
		{ID: "tutodecode", Title: "<script>", Size: 5},
	}}}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://tutodecode.t2c"+SitesPath, nil)
	s.serveSite(rec, req, "tutodecode.t2c")
	var out struct {
		Version uint64
		Sites   []siteInfo
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || out.Version != 7 || len(out.Sites) != 2 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if out.Sites[1].Address != "tutodecode.t2c" || !strings.Contains(rec.Header().Get("Content-Type"), "json") {
		t.Fatalf("%+v", out.Sites[1])
	}
	rec = httptest.NewRecorder()
	s.serveSite(rec, httptest.NewRequest("POST", "http://tutodecode.t2c"+SitesPath, nil), "tutodecode.t2c")
	if rec.Code != 405 {
		t.Fatalf("POST : %d", rec.Code)
	}
}

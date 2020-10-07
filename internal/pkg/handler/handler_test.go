package handler

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	ngsi "github.com/iot-for-tillgenglighet/ngsi-ld-golang/pkg/ngsi-ld"
)


func TestNGSI(t *testing.T) {
	contextRegistry := ngsi.NewContextRegistry()
	ctxSource := contextSource{}
	contextRegistry.Register(&ctxSource)
	router := createRequestRouter(contextRegistry)

	ts := httptest.NewServer(router.impl)
	defer ts.Close()

	// PATCH 
	// _, body := testRequest(t, ts, "PATCH", "/ngsi-ld/v1/entities/mytestentity/attrs/", nil); 
	// if body != "[]" {
	// 	t.Fatalf(body)
	// }

		// GET 
	_, body := testRequest(t, ts, "GET", "/ngsi-ld/v1/entities?attrs=temperature", nil); 
	if body != "[]" {
		t.Fatalf(body)
	}
}

func testRequest(t *testing.T, ts *httptest.Server, method, path string, body io.Reader) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, body)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}
	defer resp.Body.Close()

	return resp, string(respBody)
}

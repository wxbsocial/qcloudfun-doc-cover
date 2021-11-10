package main

import (
	"net/url"
	"strings"
	"testing"
)

func TestObjectKey(t *testing.T) {

	u := "http://coz-private-1306409624.cos.ap-guangzhou.myqcloud.com/f00122ad-3763-4081-8b8b-f665c2be6e15/share/5c9b7cb2-9982-426e-8855-429ba277afbd/pwd/%E6%B5%B7%E9%BE%9F%E4%BA%BA%E7%BE%8E2.pptx"

	objectUrl, err := url.Parse(u)
	if err != nil {
		t.Fatal(err)
	}
	//ci, _ := url.Parse("https://coz-private-1306409624.ci.ap-guangzhou.myqcloud.com")

	ciUrl := "https://" + strings.Replace(objectUrl.Host, ".cos.", ".ci.", 1)
	objectKey := strings.TrimPrefix(objectUrl.Path, "/")

	t.Logf("ciUrl:%v,objectKey:%v", ciUrl, objectKey)
}

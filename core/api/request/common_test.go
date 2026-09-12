package request

import "testing"

func TestPageInfoPageOffsetNormalizesBounds(t *testing.T) {
	var nilInfo *PageInfo
	page, size, offset := nilInfo.PageOffset()
	if page != DefaultPage || size != DefaultPageSize || offset != 0 {
		t.Fatalf("nil page info = (%d, %d, %d)", page, size, offset)
	}

	info := PageInfo{Page: MaxPage + 1, PageSize: MaxPageSize + 1}
	page, size, offset = info.PageOffset()
	if page != MaxPage || size != MaxPageSize || offset != MaxPageSize*(MaxPage-1) {
		t.Fatalf("normalized page info = (%d, %d, %d)", page, size, offset)
	}
}

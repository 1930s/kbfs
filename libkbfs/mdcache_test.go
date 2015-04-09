package libkbfs

import (
	"fmt"
	"testing"

	"code.google.com/p/gomock/gomock"
	"github.com/keybase/go-jsonw"
	libkb "github.com/keybase/client/go/libkb"
)

func mdCacheInit(t *testing.T, cap int) (
	mockCtrl *gomock.Controller, config *ConfigMock) {
	mockCtrl = gomock.NewController(t)
	config = NewConfigMock(mockCtrl)
	mdcache := NewMDCacheStandard(cap)
	config.SetMDCache(mdcache)
	return
}

func expectUserCall(u libkb.UID, config *ConfigMock) {
	userString :=
		fmt.Sprintf(`{"basics" : {"username" : "user_%s"}, "id" : "%s"}`,
			u, u)
	jsonU, _ := jsonw.Unmarshal([]byte(userString))
	user, _ := libkb.NewUser(jsonU)
	config.mockKbpki.EXPECT().GetUser(u).AnyTimes().Return(user, nil)
}

func expectUserCalls(handle *DirHandle, config *ConfigMock) {
	for _, u := range handle.Writers {
		expectUserCall(u, config)
	}
	for _, u := range handle.Readers {
		expectUserCall(u, config)
	}
}

func testMdcachePut(t *testing.T, id DirId, h *DirHandle, config *ConfigMock) {
	rmd := &RootMetadata{
		Keys: make([]DirKeys, 1, 1),
	}
	k := DirKeys{}
	rmd.Keys[0] = k

	// put the md
	expectUserCalls(h, config)
	if err := config.MDCache().Put(id, rmd); err != nil {
		t.Errorf("Got error on put on md %v: %v", id, err)
	}

	// make sure we can get it successfully
	if rmd2, err := config.MDCache().Get(id); err != nil {
		t.Errorf("Got error on get for md %v: %v", id, err)
	} else if rmd2 != rmd {
		t.Errorf("Got back unexpected metadata: %v", rmd2)
	}
}

func TestMdcachePut(t *testing.T) {
	mockCtrl, config := mdCacheInit(t, 100)
	defer mockCtrl.Finish()

	id, h, _ := newDir(config, 1, true, false)
	h.Writers = append(h.Writers, libkb.UID{0})

	testMdcachePut(t, id, h, config)
}

func TestMdcachePutPastCapacity(t *testing.T) {
	mockCtrl, config := mdCacheInit(t, 2)
	defer mockCtrl.Finish()

	id0, h0, _ := newDir(config, 1, true, false)
	h0.Writers = append(h0.Writers, libkb.UID{0})

	id1, h1, _ := newDir(config, 2, true, false)
	h1.Writers = append(h1.Writers, libkb.UID{1})

	id2, h2, _ := newDir(config, 3, true, false)
	h2.Writers = append(h2.Writers, libkb.UID{2})

	testMdcachePut(t, id0, h0, config)
	testMdcachePut(t, id1, h1, config)
	testMdcachePut(t, id2, h2, config)

	// id 0 should no longer be in the cache
	// make sure we can get it successfully
	expectUserCalls(h0, config)
	expectedErr := &NoSuchMDError{id0.String()}
	if _, err := config.MDCache().Get(id0); err == nil {
		t.Errorf("No expected error on get")
	} else if err.Error() != expectedErr.Error() {
		t.Errorf("Got unexpected error on get: %v", err)
	}
}

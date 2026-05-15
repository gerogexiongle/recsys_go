package featurestore

import "testing"

func TestMergeUserItemJSON(t *testing.T) {
	u := []byte(`{"fm_sparse":[{"k":"100","w":0.5}],"tf_dense":[0.1,0.2]}`)
	i := []byte(`{"fm_sparse":[{"k":200,"w":1}],"tf_dense":[0.9,0.8,0.7,0.6,0.5,0.4,0.3,0.2]}`)
	sp, d, err := MergeUserItemJSON(u, i)
	if err != nil {
		t.Fatal(err)
	}
	if len(sp) != 2 {
		t.Fatalf("sparse %d", len(sp))
	}
	if len(d) != 8 || d[0] != 0.9 {
		t.Fatalf("dense %+v", d)
	}
}

func TestParseUserExposure(t *testing.T) {
	u := []byte(`{"exposure":{"910005":15,"910001":1}}`)
	m := ParseUserExposure(u)
	if m[910005] != 15 {
		t.Fatalf("%+v", m)
	}
}

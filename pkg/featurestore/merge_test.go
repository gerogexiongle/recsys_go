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

func TestMergeUserItemJSON_noProfileUsesPlaceholderPath(t *testing.T) {
	sp, d, err := MergeUserItemJSON(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(sp) != 0 || len(d) != 0 {
		t.Fatalf("empty profile => empty merge sp=%d d=%d", len(sp), len(d))
	}
}

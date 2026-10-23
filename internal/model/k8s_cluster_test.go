package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestK8sClusterJSONHidesKubeconfig(t *testing.T) {
	secret := "apiVersion: v1\nusers:\n- user: {client-key-data: FIXTURE-PRIVATE-KEY}\n"
	cases := []struct {
		content string
		want    bool
	}{{secret, true}, {"", false}, {"  \n", false}}
	for _, tc := range cases {
		c := K8sCluster{Model: Model{ID: 7}, Name: "c", KubeConfigContent: tc.content}
		for _, v := range []any{c, &c, ListResp[*K8sCluster]{Items: []*K8sCluster{&c}, Total: 1}} {
			out, err := json.Marshal(v)
			if err != nil {
				t.Fatal(err)
			}
			s := string(out)
			if strings.Contains(s, "kube_config_content") || strings.Contains(s, "FIXTURE-PRIVATE-KEY") {
				t.Fatalf("kubeconfig 被序列化: %s", s)
			}
			if !strings.Contains(s, `"has_kube_config":`+map[bool]string{true: "true", false: "false"}[tc.want]) {
				t.Fatalf("has_kube_config 不正确: %s", s)
			}
			if !strings.Contains(s, `"id":7`) || !strings.Contains(s, `"name":"c"`) {
				t.Fatalf("其他字段应保持不变: %s", s)
			}
		}
	}
}

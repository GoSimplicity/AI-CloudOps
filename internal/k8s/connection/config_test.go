package connection

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
	api "k8s.io/client-go/tools/clientcmd/api"
)

func inlineConfig() *api.Config {
	return &api.Config{CurrentContext: "safe", Clusters: map[string]*api.Cluster{"safe": {Server: "https://127.0.0.1:6443"}}, AuthInfos: map[string]*api.AuthInfo{"safe": {Token: "fixture-secret"}}, Contexts: map[string]*api.Context{"safe": {Cluster: "safe", AuthInfo: "safe"}}}
}
func encode(t *testing.T, c *api.Config) string {
	t.Helper()
	b, err := clientcmd.Write(*c)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestRejectHostCapabilitiesAndUnsafeTransport(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "must-not-execute")
	cases := map[string]func(*api.Config){
		"exec": func(c *api.Config) {
			c.AuthInfos["safe"].Exec = &api.ExecConfig{Command: "sh", Args: []string{"-c", "touch " + marker}}
		},
		"provider":         func(c *api.Config) { c.AuthInfos["safe"].AuthProvider = &api.AuthProviderConfig{Name: "oidc"} },
		"token file":       func(c *api.Config) { c.AuthInfos["safe"].TokenFile = "/etc/passwd" },
		"certificate file": func(c *api.Config) { c.AuthInfos["safe"].ClientCertificate = "/etc/cert.pem" },
		"key file":         func(c *api.Config) { c.AuthInfos["safe"].ClientKey = "/etc/private.key" },
		"CA file":          func(c *api.Config) { c.Clusters["safe"].CertificateAuthority = "/etc/cert.pem" },
		"insecure":         func(c *api.Config) { c.Clusters["safe"].InsecureSkipTLSVerify = true },
		"proxy":            func(c *api.Config) { c.Clusters["safe"].ProxyURL = "http://proxy.invalid" },
		"http":             func(c *api.Config) { c.Clusters["safe"].Server = "http://127.0.0.1:8080" },
		"URL credentials":  func(c *api.Config) { c.Clusters["safe"].Server = "https://fixture-secret@example.invalid" },
		"missing cluster":  func(c *api.Config) { delete(c.Clusters, "safe") },
		"missing context":  func(c *api.Config) { c.CurrentContext = "" },
		"unused exec":      func(c *api.Config) { c.AuthInfos["unused"] = &api.AuthInfo{Exec: &api.ExecConfig{Command: "sh"}} },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			c := inlineConfig()
			change(c)
			cfg, err := Parse(encode(t, c))
			if err == nil || cfg != nil {
				t.Fatal("unsafe kubeconfig accepted")
			}
			if strings.Contains(err.Error(), "fixture-secret") {
				t.Fatal("credential exposed in diagnostic")
			}
		})
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("credential command executed")
	}
	if _, err := Parse("fixture-secret: ["); err == nil || strings.Contains(err.Error(), "fixture-secret") {
		t.Fatal("malformed input diagnostic unsafe")
	}
}

func TestInlineCredentialsAndClusterIdentity(t *testing.T) {
	first := inlineConfig()
	cfg, err := Parse(encode(t, first))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BearerToken != "fixture-secret" || cfg.Host != "https://127.0.0.1:6443" {
		t.Fatal("inline identity changed")
	}
	second := first.DeepCopy()
	second.AuthInfos["safe"].Token = "rotated-fixture"
	same, err := SameCluster(encode(t, first), encode(t, second))
	if err != nil || !same {
		t.Fatal("credential rotation moved cluster identity", err)
	}
	legacy := first.DeepCopy()
	legacy.AuthInfos["safe"].Exec = &api.ExecConfig{Command: "must-not-run"}
	same, err = SameCluster(encode(t, legacy), encode(t, second))
	if err != nil || !same {
		t.Fatal("safe replacement of legacy auth was blocked", err)
	}
	second.Clusters["safe"].Server = "https://127.0.0.1:6444"
	same, err = SameCluster(encode(t, first), encode(t, second))
	if err != nil || same {
		t.Fatal("changed API address accepted as same cluster", err)
	}
	if path := os.Getenv("CLOUDOPS_INLINE_TEST_KUBECONFIG"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal("cannot load dedicated inline fixture")
		}
		if _, err := Parse(string(data)); err != nil {
			t.Fatal("dedicated kind kubeconfig rejected", err)
		}
	}
}

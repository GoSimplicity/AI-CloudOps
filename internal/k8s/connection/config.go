// Package connection parses uploaded kubeconfig without granting host capabilities.
package connection

import (
	"bytes"
	"errors"
	"net/url"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	api "k8s.io/client-go/tools/clientcmd/api"
)

// Parse accepts only self-contained HTTPS credentials. Credential plugins belong
// to the platform operator, never to configuration submitted through the API.
func Parse(content string) (*rest.Config, error) {
	raw, err := clientcmd.Load([]byte(content))
	if err != nil || raw == nil {
		return nil, errors.New("kubeconfig格式无效")
	}
	if raw.CurrentContext == "" || raw.Contexts[raw.CurrentContext] == nil {
		return nil, errors.New("kubeconfig必须选择有效上下文")
	}
	selected := raw.Contexts[raw.CurrentContext]
	if selected.Cluster == "" || raw.Clusters[selected.Cluster] == nil {
		return nil, errors.New("kubeconfig必须选择有效集群")
	}

	for _, user := range raw.AuthInfos {
		if user == nil {
			return nil, errors.New("kubeconfig认证配置无效")
		}
		if user.Exec != nil || user.AuthProvider != nil {
			return nil, errors.New("服务器不接受kubeconfig认证插件；请提供内嵌token或客户端证书")
		}
		if user.TokenFile != "" || user.ClientCertificate != "" || user.ClientKey != "" {
			return nil, errors.New("kubeconfig不得读取服务器文件；请使用内嵌凭据")
		}
	}
	for _, cluster := range raw.Clusters {
		if cluster == nil {
			return nil, errors.New("kubeconfig集群配置无效")
		}
		if cluster.CertificateAuthority != "" {
			return nil, errors.New("kubeconfig CA必须内嵌，不得引用服务器文件")
		}
		if cluster.InsecureSkipTLSVerify {
			return nil, errors.New("服务器不接受跳过TLS验证；请提供可信CA")
		}
		if cluster.ProxyURL != "" {
			return nil, errors.New("代理由平台运行环境配置，不能通过上传的kubeconfig指定")
		}
		server, err := url.Parse(cluster.Server)
		if err != nil || server.Scheme != "https" || server.Hostname() == "" || server.User != nil || server.RawQuery != "" || server.Fragment != "" {
			return nil, errors.New("kubeconfig API地址必须为无凭据的HTTPS地址")
		}
	}
	cfg, err := clientcmd.NewDefaultClientConfig(*raw, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, errors.New("kubeconfig上下文或认证配置无效")
	}
	return cfg, nil
}

// SameCluster compares transport identity, allowing inline credential rotation
// and harmless YAML formatting changes without moving policies to another API.
func SameCluster(first, second string) (bool, error) {
	left, err := transportIdentity(first)
	if err != nil {
		return false, err
	}
	right, err := transportIdentity(second)
	if err != nil {
		return false, err
	}
	return left.Host == right.Host && left.ServerName == right.ServerName && bytes.Equal(left.CAData, right.CAData), nil
}

// Read transport metadata without loading any old authentication capability.
// This permits replacing legacy exec credentials with safe inline credentials.
func transportIdentity(content string) (*rest.Config, error) {
	raw, err := clientcmd.Load([]byte(content))
	if err != nil {
		return nil, errors.New("kubeconfig格式无效")
	}
	for name := range raw.AuthInfos {
		raw.AuthInfos[name] = &api.AuthInfo{}
	}
	data, err := clientcmd.Write(*raw)
	if err != nil {
		return nil, errors.New("kubeconfig格式无效")
	}
	return Parse(string(data))
}

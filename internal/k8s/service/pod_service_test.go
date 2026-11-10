package service

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	k8sutils "github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type podManagerFixture struct {
	manager.PodManager
	pods       []*model.K8sPod
	pod        *corev1.Pod
	err        error
	namespace  string
	name       string
	shell      string
	filePath   string
	logOptions *corev1.PodLogOptions
	ports      []model.PodPortForwardPort
	deleted    metav1.DeleteOptions
	updated    *corev1.Pod
	createdNS  string
}

func (m *podManagerFixture) GetPodList(_ context.Context, _ int, namespace string, _ metav1.ListOptions) ([]*model.K8sPod, error) {
	m.namespace = namespace
	return m.pods, m.err
}

func (m *podManagerFixture) GetPod(_ context.Context, _ int, namespace, name string) (*corev1.Pod, error) {
	m.namespace, m.name = namespace, name
	return m.pod, m.err
}

func (m *podManagerFixture) GetPodLogs(_ context.Context, _ int, _, _ string, options *corev1.PodLogOptions) (io.ReadCloser, error) {
	m.logOptions = options
	if m.err != nil {
		return nil, m.err
	}
	return io.NopCloser(strings.NewReader("line")), nil
}

func (m *podManagerFixture) PodTerminalSession(_ context.Context, _ int, _, _, _, shell string, _ *websocket.Conn) error {
	m.shell = shell
	return m.err
}

func (m *podManagerFixture) PodPortForward(_ context.Context, _ int, _, _ string, ports []model.PodPortForwardPort) error {
	m.ports = ports
	return m.err
}

func (m *podManagerFixture) UploadFileToPod(_ context.Context, _ int, _, _, _, filePath string, _ *multipart.Form) error {
	m.filePath = filePath
	return m.err
}

func (m *podManagerFixture) DownloadPodFile(_ context.Context, _ int, _, _, _, filePath string) (*k8sutils.PodFileStreamPipe, error) {
	m.filePath = filePath
	return nil, m.err
}

func (m *podManagerFixture) DeletePod(_ context.Context, _ int, _, _ string, options metav1.DeleteOptions) error {
	m.deleted = options
	return m.err
}

func (m *podManagerFixture) CreatePod(_ context.Context, _ int, namespace string, _ *corev1.Pod) (*corev1.Pod, error) {
	m.createdNS = namespace
	return m.pod, m.err
}

func (m *podManagerFixture) UpdatePod(_ context.Context, _ int, _ string, pod *corev1.Pod) (*corev1.Pod, error) {
	m.updated = pod
	return pod, m.err
}

func (m *podManagerFixture) GetPodsByNodeName(context.Context, int, string) ([]*model.K8sPod, error) {
	return m.pods, m.err
}

func TestPodListFiltersBeforePagination(t *testing.T) {
	fixture := &podManagerFixture{pods: []*model.K8sPod{
		{Name: "api-old", Status: "Running", CreatedAt: time.Unix(1, 0)},
		{Name: "batch", Status: "Running", CreatedAt: time.Unix(3, 0)},
		{Name: "api-new", Status: "Pending", CreatedAt: time.Unix(2, 0)},
		{Name: "API-mid", Status: "Running", CreatedAt: time.Unix(4, 0)},
	}}
	svc := NewPodService(fixture, zap.NewNop())
	result, err := svc.GetPodList(context.Background(), &model.GetPodListReq{
		ListReq: model.ListReq{Page: 2, Size: 1, Search: "ApI"}, ClusterID: 4, Status: "Running",
	})
	if err != nil || fixture.namespace != corev1.NamespaceAll || result.Total != 2 || len(result.Items) != 1 || result.Items[0].Name != "api-old" {
		t.Fatalf("page/filter = %+v ns=%q err=%v", result, fixture.namespace, err)
	}
	if _, err := svc.GetPodList(context.Background(), nil); err == nil {
		t.Fatal("nil list accepted")
	}
	if _, err := svc.GetPodList(context.Background(), &model.GetPodListReq{}); err == nil {
		t.Fatal("missing cluster accepted")
	}
	fixture.err = errors.New("apiserver down")
	if _, err := svc.GetPodList(context.Background(), &model.GetPodListReq{ClusterID: 4, Namespace: "ops"}); err == nil || !strings.Contains(err.Error(), "获取Pod列表失败") || fixture.namespace != "ops" {
		t.Fatalf("list error = %v", err)
	}
}

func TestPodReadAndMutationValidation(t *testing.T) {
	fixture := &podManagerFixture{pod: &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "ops"},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "example.invalid/app:test"}}},
	}}
	svc := NewPodService(fixture, zap.NewNop())
	if _, err := svc.GetPodDetails(context.Background(), nil); err == nil {
		t.Fatal("nil details accepted")
	}
	if _, err := svc.GetPodDetails(context.Background(), &model.GetPodDetailsReq{ClusterID: 1, Namespace: "ops"}); err == nil {
		t.Fatal("missing name accepted")
	}
	got, err := svc.GetPodDetails(context.Background(), &model.GetPodDetailsReq{ClusterID: 1, Namespace: "ops", Name: "app"})
	if err != nil || got.Name != "app" || got.Namespace != "ops" {
		t.Fatalf("details = %+v %v", got, err)
	}
	yaml, err := svc.GetPodYaml(context.Background(), &model.GetPodYamlReq{ClusterID: 1, Namespace: "ops", Name: "app"})
	if err != nil || !strings.Contains(yaml.YAML, "example.invalid/app:test") {
		t.Fatalf("yaml = %+v %v", yaml, err)
	}
	fixture.err = errors.New("forbidden")
	if _, err := svc.GetPodYaml(context.Background(), &model.GetPodYamlReq{ClusterID: 1, Namespace: "ops", Name: "app"}); err == nil || !strings.Contains(err.Error(), "获取Pod失败") {
		t.Fatalf("yaml error = %v", err)
	}
	fixture.err = nil
	containers, err := svc.GetPodContainers(context.Background(), &model.GetPodContainersReq{ClusterID: 1, Namespace: "ops", PodName: "app"})
	if err != nil || containers.Total != 1 || containers.Items[0].Name != "app" {
		t.Fatalf("containers = %+v %v", containers, err)
	}
	if _, err := svc.GetPodsByNodeName(context.Background(), &model.GetPodsByNodeReq{ClusterID: 1}); err == nil {
		t.Fatal("missing node accepted")
	}
	fixture.pods = []*model.K8sPod{{Name: "on-node"}}
	listed, err := svc.GetPodsByNodeName(context.Background(), &model.GetPodsByNodeReq{ClusterID: 1, NodeName: "worker-a"})
	if err != nil || listed.Total != 1 {
		t.Fatalf("node pods = %+v %v", listed, err)
	}

	grace := int64(15)
	if err := svc.DeletePod(context.Background(), &model.DeletePodReq{ClusterID: 1, Namespace: "ops", Name: "app", GracePeriodSeconds: &grace, Force: true}); err != nil {
		t.Fatal(err)
	}
	if fixture.deleted.GracePeriodSeconds == nil || *fixture.deleted.GracePeriodSeconds != 15 || fixture.deleted.PropagationPolicy == nil || *fixture.deleted.PropagationPolicy != metav1.DeletePropagationBackground {
		t.Fatalf("delete options = %+v", fixture.deleted)
	}
	fixture.err = errors.New("conflict")
	if err := svc.DeletePod(context.Background(), &model.DeletePodReq{ClusterID: 1, Namespace: "ops", Name: "app"}); err == nil || !strings.Contains(err.Error(), "删除Pod失败") {
		t.Fatalf("delete error = %v", err)
	}

	fixture.err = nil
	if err := svc.UpdatePod(context.Background(), &model.UpdatePodReq{ClusterID: 1, Namespace: "ops", Name: "app", Labels: map[string]string{"owner": "team-a"}}); err != nil {
		t.Fatal(err)
	}
	if fixture.updated.Labels["owner"] != "team-a" || fixture.updated.Namespace != "ops" {
		t.Fatalf("updated pod = %+v", fixture.updated)
	}
	if err := svc.CreatePod(context.Background(), &model.CreatePodReq{ClusterID: 1, Name: "app", Namespace: "ops"}); err == nil {
		t.Fatal("pod without containers accepted")
	}
	if err := svc.CreatePod(context.Background(), &model.CreatePodReq{
		ClusterID: 1, Name: "app", Namespace: "ops",
		Containers: []model.CreatePodContainer{{Name: "app", Image: "example.invalid/app:test"}},
	}); err != nil || fixture.createdNS != "ops" {
		t.Fatalf("create = %v ns=%s", err, fixture.createdNS)
	}
}

func TestPodLogsPortsAndExecValidation(t *testing.T) {
	fixture := &podManagerFixture{}
	svc := NewPodService(fixture, zap.NewNop())
	if _, err := svc.GetPodLogs(context.Background(), nil); err == nil {
		t.Fatal("nil logs accepted")
	}
	if _, err := svc.GetPodLogs(context.Background(), &model.GetPodLogsReq{ClusterID: 1, Namespace: "ops", PodName: "app", SinceTime: "yesterday"}); err == nil || !strings.Contains(err.Error(), "时间参数格式错误") {
		t.Fatalf("bad time = %v", err)
	}
	since := int64(30)
	reader, err := svc.GetPodLogs(context.Background(), &model.GetPodLogsReq{
		ClusterID: 1, Namespace: "ops", PodName: "app", Container: "app", SinceTime: "2020-01-02T03:04:05Z", SinceSeconds: &since, Timestamps: true,
	})
	if err != nil || fixture.logOptions == nil || fixture.logOptions.Container != "app" || fixture.logOptions.SinceTime == nil || fixture.logOptions.SinceTime.Time.UTC().Format(time.RFC3339) != "2020-01-02T03:04:05Z" {
		t.Fatalf("log options = %+v %v", fixture.logOptions, err)
	}
	body, _ := io.ReadAll(reader)
	_ = reader.Close()
	if string(body) != "line" {
		t.Fatalf("log body = %q", body)
	}
	fixture.err = errors.New("previous terminated container app not found")
	_, err = svc.GetPodLogs(context.Background(), &model.GetPodLogsReq{ClusterID: 1, Namespace: "ops", PodName: "app"})
	var previous *PodLogsPreviousNotFoundError
	if !errors.As(err, &previous) || previous.Unwrap() == nil {
		t.Fatalf("previous logs = %v", err)
	}
	fixture.err = errors.New("forbidden")
	if _, err := svc.GetPodLogs(context.Background(), &model.GetPodLogsReq{ClusterID: 1, Namespace: "ops", PodName: "app"}); err == nil || !strings.Contains(err.Error(), "获取Pod日志失败") {
		t.Fatalf("log error = %v", err)
	}

	if err := svc.PodExec(context.Background(), &model.PodExecReq{ClusterID: 1, Namespace: "ops", PodName: "app", Shell: "sh"}, nil); err == nil {
		t.Fatal("nil websocket accepted")
	}
	fixture.err = errors.New("exec denied")
	conn := serviceWebsocket(t)
	err = svc.PodExec(context.Background(), &model.PodExecReq{ClusterID: 1, Namespace: "ops", PodName: "app", Container: "app", Shell: "bash -c 'echo hi'"}, conn)
	var shellErr *base.BusinessError
	if err == nil || fixture.shell != "" || !errors.As(err, &shellErr) || !errors.Is(shellErr.Code, constants.ErrInvalidParam) || !strings.Contains(err.Error(), "不支持的shell类型") {
		t.Fatalf("exec shell = %v forwarded=%q", err, fixture.shell)
	}
	err = svc.PodExec(context.Background(), &model.PodExecReq{ClusterID: 1, Namespace: "ops", PodName: "app", Container: "app", Shell: "bash"}, conn)
	if err == nil || fixture.shell != "bash" || !strings.Contains(err.Error(), "exec denied") {
		t.Fatalf("exec = %v shell=%q", err, fixture.shell)
	}

	if err := svc.PodPortForward(context.Background(), &model.PodPortForwardReq{ClusterID: 1, Namespace: "ops", PodName: "app"}); err == nil {
		t.Fatal("empty ports accepted")
	}
	if err := svc.PodPortForward(context.Background(), &model.PodPortForwardReq{ClusterID: 1, Namespace: "ops", PodName: "app", Ports: []model.PodPortForwardPort{{LocalPort: 0, RemotePort: 80}}}); err == nil {
		t.Fatal("local port 0 accepted")
	}
	if err := svc.PodPortForward(context.Background(), &model.PodPortForwardReq{ClusterID: 1, Namespace: "ops", PodName: "app", Ports: []model.PodPortForwardPort{{LocalPort: 8080, RemotePort: 65536}}}); err == nil {
		t.Fatal("remote port 65536 accepted")
	}
	fixture.err = nil
	if err := svc.PodPortForward(context.Background(), &model.PodPortForwardReq{ClusterID: 1, Namespace: "ops", PodName: "app", Ports: []model.PodPortForwardPort{{LocalPort: 8080, RemotePort: 80}}}); err != nil || len(fixture.ports) != 1 {
		t.Fatalf("forward = %v %+v", err, fixture.ports)
	}
}

func TestPodFilePathsAndYAMLIdentity(t *testing.T) {
	fixture := &podManagerFixture{}
	svc := NewPodService(fixture, zap.NewNop())
	form := &multipart.Form{}
	rejected := []string{
		"..",
		"../etc/passwd",
		"/tmp/../etc/passwd",
		"foo/../../etc/shadow",
		"/var/log/../../etc/passwd",
		"a\x00b",
		"foo\nbar",
		"~/secret",
	}
	for _, path := range rejected {
		fixture.filePath = ""
		err := svc.PodFileUpload(context.Background(), &model.PodFileUploadReq{
			ClusterID: 1, Namespace: "ops", PodName: "app", ContainerName: "app", FilePath: path,
		}, form)
		var biz *base.BusinessError
		if err == nil || fixture.filePath != "" || !errors.As(err, &biz) || !errors.Is(biz.Code, constants.ErrInvalidParam) {
			t.Fatalf("upload %q = %v path=%q", path, err, fixture.filePath)
		}
		_, _, err = svc.PodFileDownload(context.Background(), &model.PodFileDownloadReq{
			ClusterID: 1, Namespace: "ops", PodName: "app", ContainerName: "app", FilePath: path,
		})
		if err == nil || fixture.filePath != "" || !errors.As(err, &biz) || !errors.Is(biz.Code, constants.ErrInvalidParam) {
			t.Fatalf("download %q = %v path=%q", path, err, fixture.filePath)
		}
	}

	fixture.err = errors.New("upload denied")
	err := svc.PodFileUpload(context.Background(), &model.PodFileUploadReq{
		ClusterID: 1, Namespace: "ops", PodName: "app", ContainerName: "app", FilePath: "foo..bar",
	}, form)
	if err == nil || fixture.filePath != "foo..bar" || !strings.Contains(err.Error(), "上传文件到Pod失败") {
		t.Fatalf("dotted name = %v path=%q", err, fixture.filePath)
	}
	for _, shell := range []string{"zsh", "fish", "ash", "/bin/bash", "bash -c id"} {
		conn := serviceWebsocket(t)
		err = svc.PodExec(context.Background(), &model.PodExecReq{
			ClusterID: 1, Namespace: "ops", PodName: "app", Container: "app", Shell: shell,
		}, conn)
		var biz *base.BusinessError
		if err == nil || !errors.As(err, &biz) || !errors.Is(biz.Code, constants.ErrInvalidParam) {
			t.Fatalf("shell %q = %v", shell, err)
		}
	}
}

func TestPodFileNameAndYAMLChecks(t *testing.T) {
	fixture := &podManagerFixture{}
	svc := NewPodService(fixture, zap.NewNop())
	form := &multipart.Form{}
	if err := svc.PodFileUpload(context.Background(), &model.PodFileUploadReq{ClusterID: 1, Namespace: "ops", PodName: "app", ContainerName: "app"}, form); err == nil {
		t.Fatal("empty path accepted")
	}
	fixture.err = errors.New("upload denied")
	err := svc.PodFileUpload(context.Background(), &model.PodFileUploadReq{
		ClusterID: 1, Namespace: "ops", PodName: "app", ContainerName: "app", FilePath: "/tmp/app",
	}, form)
	if err == nil || fixture.filePath != "/tmp/app" || !strings.Contains(err.Error(), "上传文件到Pod失败") {
		t.Fatalf("upload = %v path=%s", err, fixture.filePath)
	}
	if _, _, err := svc.PodFileDownload(context.Background(), &model.PodFileDownloadReq{ClusterID: 1, Namespace: "ops", PodName: "app", ContainerName: "app"}); err == nil {
		t.Fatal("empty download path accepted")
	}
	fixture.err = nil
	_, name, err := svc.PodFileDownload(context.Background(), &model.PodFileDownloadReq{
		ClusterID: 1, Namespace: "ops", PodName: "app", ContainerName: "app", FilePath: "/var/log/app.log",
	})
	if err != nil || name != "app.log" || fixture.filePath != "/var/log/app.log" {
		t.Fatalf("download name = %q path=%s err=%v", name, fixture.filePath, err)
	}
	_, name, err = svc.PodFileDownload(context.Background(), &model.PodFileDownloadReq{
		ClusterID: 1, Namespace: "ops", PodName: "app", ContainerName: "app", FilePath: "/",
	})
	if err != nil || name != "download" {
		t.Fatalf("root download name = %q %v", name, err)
	}

	if err := svc.CreatePodByYaml(context.Background(), &model.CreatePodByYamlReq{ClusterID: 1, YAML: "{["}); err == nil || !strings.Contains(err.Error(), "解析YAML失败") {
		t.Fatalf("bad yaml = %v", err)
	}
	missingName := "apiVersion: v1\nkind: Pod\nmetadata:\n  namespace: ops\nspec:\n  containers:\n  - name: app\n    image: example.invalid/app:test\n"
	if err := svc.CreatePodByYaml(context.Background(), &model.CreatePodByYamlReq{ClusterID: 1, YAML: missingName}); err == nil || !strings.Contains(err.Error(), "必须指定name") {
		t.Fatalf("missing name = %v", err)
	}
	yaml := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: app\nspec:\n  containers:\n  - name: app\n    image: example.invalid/app:test\n"
	if err := svc.CreatePodByYaml(context.Background(), &model.CreatePodByYamlReq{ClusterID: 1, YAML: yaml}); err != nil || fixture.createdNS != "default" {
		t.Fatalf("yaml create = %v ns=%s", err, fixture.createdNS)
	}
	mismatched := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: other\n  namespace: other\nspec:\n  containers:\n  - name: app\n    image: example.invalid/app:test\n"
	fixture.updated = nil
	if err := svc.UpdatePodByYaml(context.Background(), &model.UpdatePodByYamlReq{ClusterID: 1, Namespace: "ops", Name: "app", YAML: mismatched}); err == nil || !strings.Contains(err.Error(), "不一致") || fixture.updated != nil {
		t.Fatalf("identity mismatch = %v", err)
	}
	same := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: app\n  namespace: ops\nspec:\n  containers:\n  - name: app\n    image: example.invalid/app:test\n"
	fixture.err = errors.New("conflict")
	if err := svc.UpdatePodByYaml(context.Background(), &model.UpdatePodByYamlReq{ClusterID: 1, Namespace: "ops", Name: "app", YAML: same}); err == nil || !strings.Contains(err.Error(), "更新Pod失败") {
		t.Fatalf("yaml update = %v", err)
	}
}

func serviceWebsocket(t *testing.T) *websocket.Conn {
	t.Helper()
	accepted := make(chan *websocket.Conn, 1)
	web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err == nil {
			accepted <- conn
		}
	}))
	t.Cleanup(web.Close)
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(web.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	select {
	case serverConn := <-accepted:
		t.Cleanup(func() { _ = serverConn.Close() })
	case <-time.After(2 * time.Second):
		t.Fatal("websocket did not accept")
	}
	return conn
}

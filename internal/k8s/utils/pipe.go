package utils

import (
	"context"
	"fmt"
	"io"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"
)

// PodFileStreamPipe pod内文件下载专用，简化实现
type PodFileStreamPipe struct {
	namespace     string
	podName       string
	containerName string
	filePath      string

	ctx context.Context

	config       *rest.Config
	client       kubernetes.Interface
	readerStream *io.PipeReader
	writerStream *io.PipeWriter

	cancelFunc context.CancelFunc
	done       chan struct{}
}

func NewPodFileStreamPipe(ctx context.Context, config *rest.Config, client kubernetes.Interface,
	namespace, pod, container, filePath string) (*PodFileStreamPipe, error) {

	// 创建可取消的上下文
	downloadCtx, cancel := context.WithCancel(ctx)

	pfs := &PodFileStreamPipe{
		ctx:    downloadCtx,
		config: config,
		client: client,

		filePath:      filePath,
		namespace:     namespace,
		podName:       pod,
		containerName: container,

		cancelFunc: cancel,
		done:       make(chan struct{}),
	}

	// 启动文件下载
	if err := pfs.startFileDownload(); err != nil {
		cancel()
		return nil, fmt.Errorf("启动文件下载失败: %w", err)
	}

	return pfs, nil
}

// startFileDownload 启动文件下载流
func (pfs *PodFileStreamPipe) startFileDownload() error {
	// 创建管道
	pfs.readerStream, pfs.writerStream = io.Pipe()

	// 构建tar命令，直接打包文件或目录
	cmd := fmt.Sprintf("tar cf - '%s' 2>/dev/null || echo 'Error: file not found'", pfs.filePath)

	// 构建exec请求
	req := pfs.client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pfs.podName).
		Namespace(pfs.namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: pfs.containerName,
			Command:   []string{"sh", "-c", cmd},
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	// 创建执行器
	exec, err := remotecommand.NewSPDYExecutor(pfs.config, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("创建执行器失败: %w", err)
	}

	// 启动异步执行
	go pfs.executeDownload(exec)

	return nil
}

// executeDownload 执行下载
func (pfs *PodFileStreamPipe) executeDownload(exec remotecommand.Executor) {
	defer close(pfs.done)
	defer pfs.writerStream.Close()
	defer pfs.cancelFunc()

	// 创建带超时的上下文 - 10分钟超时
	timeoutCtx, cancel := context.WithTimeout(pfs.ctx, 10*time.Minute)
	defer cancel()

	// 执行命令并将输出写入管道
	err := exec.StreamWithContext(timeoutCtx, remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: pfs.writerStream,
		Stderr: pfs.writerStream, // 将错误输出也写入同一个流
		Tty:    false,
	})

	if err != nil {
		// 将错误写入流
		errorMsg := fmt.Sprintf("文件下载失败: %v", err)
		pfs.writerStream.Write([]byte(errorMsg))
		pfs.writerStream.CloseWithError(fmt.Errorf("文件下载失败: %w", err))
	}
}

// Read 实现io.Reader接口
func (pfs *PodFileStreamPipe) Read(p []byte) (int, error) {
	return pfs.readerStream.Read(p)
}

// Close 关闭管道和资源
func (pfs *PodFileStreamPipe) Close() error {
	// 取消上下文
	if pfs.cancelFunc != nil {
		pfs.cancelFunc()
	}

	// 等待下载协程结束（带超时）
	select {
	case <-pfs.done:
		// 下载协程已结束
	case <-time.After(5 * time.Second):
		// 超时，强制关闭
	}

	// 关闭读取端
	var readError error
	if pfs.readerStream != nil {
		readError = pfs.readerStream.Close()
	}

	// 关闭写入端
	var writeError error
	if pfs.writerStream != nil {
		writeError = pfs.writerStream.Close()
	}

	// 返回第一个遇到的错误
	if readError != nil {
		return readError
	}
	return writeError
}

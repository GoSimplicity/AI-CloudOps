package utils

import (
	"context"
	"fmt"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"
	"strconv"
	"strings"
)

// PodFileStreamPipe pod内文件下载专用 实现零拷贝
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

	bytesRead uint64
	size      uint64
}

func NewPodFileStreamPipe(ctx context.Context, config *rest.Config, client kubernetes.Interface,
	namespace, pod, container, filePath string) (*PodFileStreamPipe, error) {

	pfs := &PodFileStreamPipe{
		ctx:    ctx,
		config: config,
		client: client,

		filePath:      filePath,
		namespace:     namespace,
		podName:       pod,
		containerName: container,
	}

	if err := pfs.fetchTotalTarSize(); err != nil {
		return nil, err
	}
	if err := pfs.startReadingFromOffset(0); err != nil {
		return nil, err
	}

	return pfs, nil
}

// fetchTotalTarSize 获取 tar 流总字节数
func (pfs *PodFileStreamPipe) fetchTotalTarSize() error {
	req := pfs.client.
		CoreV1().
		RESTClient().Post().
		Resource("pods").
		Name(pfs.podName).
		Namespace(pfs.namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: pfs.containerName,
			Command:   []string{"sh", "-c", fmt.Sprintf("tar cf - %s | wc -c", pfs.filePath)},
			Stdout:    true,
			Stdin:     true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(pfs.config, "POST", req.URL())
	if err != nil {
		return err
	}
	reader, writer := io.Pipe()
	go func() {
		defer writer.Close()
		if err := exec.StreamWithContext(pfs.ctx, remotecommand.StreamOptions{
			Stdin:             nil,
			Stdout:            writer,
			Stderr:            nil,
			TerminalSizeQueue: nil,
		}); err != nil {
			writer.CloseWithError(err)
		}
	}()
	output, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	size, err := strconv.ParseUint(strings.TrimSpace(string(output)), 10, 64)
	if err != nil {
		return err
	}
	pfs.size = size
	return nil
}

// startReadingFromOffset 从偏移量读取
func (pfs *PodFileStreamPipe) startReadingFromOffset(offset uint64) error {
	pfs.readerStream, pfs.writerStream = io.Pipe()
	restClient := pfs.client.CoreV1().RESTClient()
	req := restClient.Post().
		Resource("pods").
		Name(pfs.podName).
		Namespace(pfs.namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: pfs.containerName,
			Command:   []string{"sh", "-c", fmt.Sprintf("tar cf - %s | tail -c+%d", pfs.filePath, offset+1)},
			Stdout:    true,
		}, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(pfs.config, "POST", req.URL())
	if err != nil {
		return err
	}
	go func() {
		defer pfs.writerStream.Close()
		if err := exec.StreamWithContext(pfs.ctx, remotecommand.StreamOptions{
			Stdin:             nil,
			Stdout:            pfs.writerStream,
			Stderr:            nil,
			TerminalSizeQueue: nil,
		}); err != nil {
			pfs.writerStream.CloseWithError(err)
		}
	}()
	return nil
}

func (pfs *PodFileStreamPipe) Read(p []byte) (int, error) {
	n, err := pfs.readerStream.Read(p)
	if err != nil {
		if pfs.bytesRead == pfs.size {
			return n, io.EOF
		}
		return n, pfs.startReadingFromOffset(pfs.bytesRead + 1)
	}
	pfs.bytesRead += uint64(n)
	return n, nil
}

func (pfs *PodFileStreamPipe) Close() error {
	var readError, writerError error
	if pfs.readerStream != nil {
		readError = pfs.readerStream.Close()
	}
	if pfs.writerStream != nil {
		writerError = pfs.writerStream.Close()
	}
	if readError != nil {
		return readError
	}
	return writerError
}

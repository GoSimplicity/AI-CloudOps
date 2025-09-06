package apply

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils/apply/patcher"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubectl/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//
//type Apply interface {
//	Apply(content io.Reader) error
//}
//
//type applier struct {
//	client      *kubernetes.Clientset
//	namespace   string
//	getter      *GetterFactory
//	out, errOut io.Writer
//	kubeConfig  *rest.Config
//}
//
//func NewApplier(ctx context.Context, namespace string, kubeConfig *rest.Config) (Apply, error) {
//	if kubeConfig == nil {
//		return nil, errors.New("kube config is nil")
//	}
//	if len(namespace) == 0 {
//		namespace = metav1.NamespaceDefault
//	}
//	return &applier{
//		namespace:  namespace,
//		kubeConfig: kubeConfig,
//		getter:     NewGetterFactory(ctx, kubeConfig),
//
//		out:    os.Stdout,
//		errOut: os.Stderr,
//	}, nil
//}
//
//func (a *applier) makeBuilder(opt *BuilderOptions) *resource.Builder {
//	if opt == nil {
//		opt = NewBuilderOptions()
//	}
//
//	namespace := a.namespace
//	if opt.Namespace != "" {
//		namespace = opt.Namespace
//	}
//
//	// 获取 validator（可选）
//	var validator validation.Schema
//	if len(opt.Validate) > 0 {
//		validator, _ = a.getter.Validator(opt.Validate)
//	}
//
//	builder := a.getter.NewBuilder().
//		Unstructured().
//		ContinueOnError().
//		NamespaceParam(namespace).
//		DefaultNamespace()
//
//	// 仅当 validator 存在时才设置 Schema
//	if validator != nil {
//		builder = builder.Schema(validator)
//	}
//
//	return builder
//}
//
//var warningNoLastAppliedConfigAnnotation = "Warning: resource %s is missing the %s annotation which is required by apply. The missing annotation will be patched automatically.\n"
//
//func (a *applier) Apply(content io.Reader) error {
//
//	inofs, err := a.ResourceForReader(content, nil)
//	if err != nil {
//		return err
//	}
//	var errs []error
//	for _, info := range inofs {
//		if err := a.ApplyResourceOne(info); err != nil {
//			errs = append(errs, err)
//		}
//	}
//	return k8serrors.NewAggregate(errs)
//
//}
//
//func (a *applier) ResourceForReader(content io.Reader, opt *BuilderOptions) ([]*resource.Info, error) {
//
//	result := a.makeBuilder(opt).
//		Stream(content, "").
//		Flatten().
//		Do()
//
//	return result.Infos()
//}
//
//func (a *applier) ApplyResourceOne(info *resource.Info) error {
//	helper := resource.NewHelper(info.Client, info.Mapping)
//
//	// 如果不存在，就直接创建
//	if err := info.Get(); err != nil {
//		if !IsNotFound(err) {
//			return fmt.Errorf("error retrieving current configuration of\n%s\nfrom server: %v", info.String(), err)
//		}
//		if err := util.CreateApplyAnnotation(info.Object, unstructured.UnstructuredJSONScheme); err != nil {
//			return err
//		}
//		if u, ok := info.Object.(*unstructured.Unstructured); ok {
//			pruneUnstructured(u)
//		}
//		obj, err := helper.Create(info.Namespace, true, info.Object)
//		if err != nil {
//			return err
//		}
//		return info.Refresh(obj, true)
//	}
//
//	// 检查 last-applied annotation
//	metadata, _ := meta.Accessor(info.Object)
//	annotationMap := metadata.GetAnnotations()
//
//	if _, ok := annotationMap[corev1.LastAppliedConfigAnnotation]; !ok {
//		return errors.New(fmt.Sprintf(warningNoLastAppliedConfigAnnotation,
//			info.ObjectName(), info.Mapping.GroupVersionKind.Kind))
//	}
//
//	// 进行裁剪，避免 status / managedFields 干扰
//	if u, ok := info.Object.(*unstructured.Unstructured); ok {
//		pruneUnstructured(u)
//	}
//
//	// 计算 patch
//	patcher, err := newPatcher(info, resource.NewHelper(info.Client, info.Mapping))
//	if err != nil {
//		return err
//	}
//
//	modified, err := util.GetModifiedConfiguration(info.Object,
//		true, unstructured.UnstructuredJSONScheme)
//	if err != nil {
//		return fmt.Errorf("retrieving modified configuration from \n%s\n: %v", info.String(), err)
//	}
//
//	patchBytes, patchedObject, err := patcher.Patch(info.Object, modified,
//		info.Source, info.Namespace, info.Name, a.errOut)
//	if err != nil {
//		return fmt.Errorf("failed to applying patch configuration:\n%s\nto:\n%v, err: %v", patchBytes, info.String(), err)
//	}
//
//	return info.Refresh(patchedObject, true)
//}
//
//func IsNotFound(err error) bool {
//	reason, code := reasonAndCodeForError(err)
//	if reason == metav1.StatusReasonNotFound {
//		return true
//	}
//	if _, ok := knownReasons[reason]; ok || code == http.StatusNotFound {
//		return true
//	}
//	return false
//}
//
//type APIStatus interface {
//	Status() metav1.Status
//}
//
//func reasonAndCodeForError(err error) (metav1.StatusReason, int32) {
//	if status, ok := err.(APIStatus); ok || errors.As(err, &status) {
//		return status.Status().Reason, status.Status().Code
//	}
//	return metav1.StatusReasonUnknown, 0
//}
//
//var knownReasons = map[metav1.StatusReason]struct{}{
//	metav1.StatusReasonUnknown:               {},
//	metav1.StatusReasonUnauthorized:          {},
//	metav1.StatusReasonForbidden:             {},
//	metav1.StatusReasonNotFound:              {},
//	metav1.StatusReasonAlreadyExists:         {},
//	metav1.StatusReasonConflict:              {},
//	metav1.StatusReasonGone:                  {},
//	metav1.StatusReasonInvalid:               {},
//	metav1.StatusReasonServerTimeout:         {},
//	metav1.StatusReasonStoreReadError:        {},
//	metav1.StatusReasonTimeout:               {},
//	metav1.StatusReasonTooManyRequests:       {},
//	metav1.StatusReasonBadRequest:            {},
//	metav1.StatusReasonMethodNotAllowed:      {},
//	metav1.StatusReasonNotAcceptable:         {},
//	metav1.StatusReasonRequestEntityTooLarge: {},
//	metav1.StatusReasonUnsupportedMediaType:  {},
//	metav1.StatusReasonInternalError:         {},
//	metav1.StatusReasonExpired:               {},
//	metav1.StatusReasonServiceUnavailable:    {},
//}
//
//// pruneUnstructured 尝试裁剪掉不应该由用户管理的字段
//func pruneUnstructured(u *unstructured.Unstructured) {
//	unstructured.RemoveNestedField(u.Object, "metadata", "managedFields")
//	unstructured.RemoveNestedField(u.Object, "metadata", "creationTimestamp")
//	unstructured.RemoveNestedField(u.Object, "status")
//}

func CreateOrPatchUnstructured(ctx context.Context, u *unstructured.Unstructured, cl client.Client) error {
	return createOrPatchObject(ctx, u, cl)
}

func createOrPatchObject(ctx context.Context, modified client.Object, cl client.Client) error {
	c := modified.DeepCopyObject().(client.Object)
	found, err := patcher.GetResource(modified.GetNamespace(), modified.GetName(), c, cl)
	if err != nil {
		return err
	} else if !found {
		return createObject(ctx, modified, cl)
	}

	// fill gvk in case it is missing.
	// for objects retrieved from APIServer, gvk is not set, so we need to set it
	// for objects in cache, gvk is set, nothing will be changed here
	gvk := modified.GetObjectKind().GroupVersionKind()
	current, err := patcher.SetGroupVersionKind(c, &gvk)
	if err != nil {
		return err
	}

	patchBytes, patchType, err := patcher.GeneratePatchBytes(current, modified)
	if err != nil {
		return err
	}

	if len(patchBytes) == 0 || patchType == "" || string(patchBytes) == "{}" {
		return nil
	}

	return patchObjectWithType(ctx, current.(client.Object), patchBytes, patchType, cl)
}

func createObject(ctx context.Context, obj client.Object, cl client.Client) error {
	// always add the last-applied-configuration annotation
	err := util.CreateApplyAnnotation(obj, unstructured.UnstructuredJSONScheme)
	if err != nil {
		return err
	}
	return cl.Create(ctx, obj)
}

func patchObjectWithType(ctx context.Context, obj client.Object, patchBytes []byte, patchType types.PatchType, cl client.Client) error {
	return cl.Patch(ctx, obj, client.RawPatch(patchType, patchBytes))
}

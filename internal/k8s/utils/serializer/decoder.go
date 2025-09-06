package decoder

import (
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"sync"
)

type decoder struct {
	typedDeserializer      runtime.Decoder
	unstructuredSerializer runtime.Serializer
}

var (
	d          *decoder
	decodeOnce sync.Once
)

func NewDecoder() *decoder {
	// TODO: 后期根据集群动态添加
	if d == nil {
		decodeOnce.Do(func() {
			schemes := runtime.NewScheme()
			d = &decoder{
				typedDeserializer:      serializer.NewCodecFactory(schemes).UniversalDeserializer(),
				unstructuredSerializer: yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme),
			}
		})
	}
	return d
}

func (d *decoder) YamlToUnstructured(manifest []byte) (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{}
	decoded, _, err := d.unstructuredSerializer.Decode(manifest, nil, obj)

	u, ok := decoded.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("object is not an Unstructured")
	}

	return u, err
}

func (d *decoder) YamlOrJsonToObject(manifest []byte) (runtime.Object, error) {
	obj, _, err := d.typedDeserializer.Decode(manifest, nil, nil)
	return obj, err
}

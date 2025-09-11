package query

const (
	FieldCreationTimeStamp = "creationTimestamp"
	FieldStatus            = "status"
	FieldName              = "name"
	FieldSearch            = "search"
	FieldNameAndAlias      = "nameAndAlias"
	FieldNames             = "names"
	FieldUID               = "uid"
	FieldLabel             = "label"
	FieldAnnotation        = "annotation"
	FieldNamespace         = "namespace"
	FieldOwnerReference    = "ownerReference"
	FieldOwnerKind         = "ownerKind"

	FieldCreateTime          = "createTime"
	FieldLastUpdateTimestamp = "lastUpdateTimestamp"
	FieldUpdateTime          = "updateTime"
)

type (
	Field string
	Value string
)

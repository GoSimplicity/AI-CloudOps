package query

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	ParameterName          = "name"
	ParameterLabelSelector = "labelSelector"
	ParameterFieldSelector = "fieldSelector"
	ParameterPage          = "page"
	ParameterLimit         = "limit"
	ParameterOrderBy       = "sortBy"
	ParameterAscending     = "ascending"

	DefaultLimit = -1
	DefaultPage  = 1
)

type Pagination struct {
	Limit  int
	Offset int
}

var DefaultPagination = newPagination(-1, 0)

func (p *Pagination) GetValidPagination(total int) (startIndex, endIndex int) {
	// no pagination
	if p.Limit == DefaultPagination.Limit {
		return 0, total
	}

	// out of range
	if p.Limit < 0 || p.Offset < 0 || p.Offset > total {
		return 0, 0
	}

	startIndex = p.Offset
	endIndex = startIndex + p.Limit

	if endIndex > total {
		endIndex = total
	}

	return startIndex, endIndex
}

func newPagination(limit int, offset int) *Pagination {
	return &Pagination{
		Limit:  limit,
		Offset: offset,
	}
}

type Filter struct {
	Field Field `json:"field"`
	Value Value `json:"value"`
}

type Query struct {
	Pagination *Pagination
	SortBy     Field
	Ascending  bool

	Filters       map[Field]Value
	LabelSelector string
}

func (q *Query) Selector() labels.Selector {
	if selector, err := labels.Parse(q.LabelSelector); err != nil {
		return labels.Everything()
	} else {
		return selector
	}
}

func New() *Query {
	return &Query{
		Pagination: &Pagination{},
		SortBy:     "",
		Ascending:  false,
	}
}
func (q *Query) AppendLabelSelector(ls map[string]string) error {
	//labels.Set(req.Labels).AsSelector().String()
	labelsMap, err := labels.ConvertSelectorToLabelsMap(q.LabelSelector)
	if err != nil {
		return err
	}
	q.LabelSelector = labels.Merge(labelsMap, ls).String()
	return nil
}

func ParseQueryWithParameters(ctx *gin.Context) *Query {
	query := New()
	limit := GetDefaultNumber(ctx, ParameterLimit, DefaultLimit)
	page := GetDefaultNumber(ctx, ParameterPage, DefaultPage)

	query.Pagination = newPagination(limit, (page-1)*limit)

	query.SortBy = Field(GetDefaultString(ctx, ParameterOrderBy, FieldCreationTimeStamp))

	ascending, err := strconv.ParseBool(GetDefaultString(ctx, ParameterAscending, "false"))

	if err != nil {
		query.Ascending = false
	} else {
		query.Ascending = ascending
	}

	for key, values := range ctx.Request.URL.Query() {
		if !hasString([]string{ParameterPage, ParameterLimit, ParameterOrderBy, ParameterAscending, ParameterLabelSelector}, key) {
			value := ""
			if len(values) > 0 {
				value = values[0]
			}
			query.Filters[Field(key)] = Value(value)
		}
	}
	return query
}

func GetDefaultNumber(c *gin.Context, key string, defaultVal int) int {
	valStr := c.Query(key)
	if len(valStr) == 0 {
		return defaultVal
	}

	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}

func GetDefaultString(c *gin.Context, key string, defaultVal string) string {
	valStr := c.Query(key)
	if len(valStr) == 0 {
		return defaultVal
	}
	return defaultVal
}

func hasString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func DefaultObjectMetaFilter(item metav1.ObjectMeta, filter Filter) bool {
	switch filter.Field {
	case FieldNames:
		for _, name := range strings.Split(string(filter.Value), ",") {
			if item.Name == name {
				return true
			}
		}
		return false
	// /namespaces?page=1&limit=10&name=default
	case FieldName:
		return strings.Contains(item.GetName(), string(filter.Value))
		// /namespaces?page=1&limit=10&uid=a8a8d6cf-f6a5-4fea-9c1b-e57610115706
	case FieldUID:
		return strings.Compare(string(item.UID), string(filter.Value)) == 0
		// /deployments?page=1&limit=10&namespace=kubesphere-system
	case FieldNamespace:
		return strings.Compare(item.Namespace, string(filter.Value)) == 0
		// /namespaces?page=1&limit=10&ownerReference=a8a8d6cf-f6a5-4fea-9c1b-e57610115706
	case FieldOwnerReference:
		for _, ownerReference := range item.OwnerReferences {
			if strings.Compare(string(ownerReference.UID), string(filter.Value)) == 0 {
				return true
			}
		}
		return false
		// /namespaces?page=1&limit=10&ownerKind=Workspace
	case FieldOwnerKind:
		for _, ownerReference := range item.OwnerReferences {
			if strings.Compare(ownerReference.Kind, string(filter.Value)) == 0 {
				return true
			}
		}
		return false
		// /namespaces?page=1&limit=10&annotation=openpitrix_runtime
	case FieldAnnotation:
		return labelMatch(item.Annotations, string(filter.Value))
		// /namespaces?page=1&limit=10&label=kubesphere.io/workspace:system-workspace
	case FieldLabel:
		return labelMatch(item.Labels, string(filter.Value))
	default:
		return true
	}
}

func DefaultObjectMetaCompare(left, right metav1.ObjectMeta, sortBy Field) bool {
	switch sortBy {
	// ?sortBy=name
	case FieldName:
		return strings.Compare(left.Name, right.Name) > 0
	//	?sortBy=creationTimestamp
	default:
		fallthrough
	case FieldCreateTime:
		fallthrough
	case FieldCreationTimeStamp:
		// compare by name if creation timestamp is equal
		if left.CreationTimestamp.Equal(&right.CreationTimestamp) {
			return strings.Compare(left.Name, right.Name) > 0
		}
		return left.CreationTimestamp.After(right.CreationTimestamp.Time)
	}
}

func labelMatch(m map[string]string, filter string) bool {
	labelSelector, err := labels.Parse(filter)
	if err != nil {
		return false
	}
	return labelSelector.Matches(labels.Set(m))
}

package base

import (
	"errors"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// These fields are assigned by handlers from route parameters or authenticated
// claims before binding. Only an already populated value is authoritative;
// pagination, filters and other client-editable defaults remain bindable.
var authoritativeFields = []string{
	"ID", "ClusterID", "UserID", "OperatorID", "OperatorName", "CreateUserID",
	"CreateUserName", "CreatedBy", "CreatedByName", "Namespace", "Name",
	"NodeName", "PodName", "ContainerName", "ServiceName", "ServiceAccountName",
	"DeploymentName", "InstanceID", "NodeID", "OnDutyGroupID",
}

func BindRequest(ctx *gin.Context, req interface{}) error {
	value := reflect.ValueOf(req)
	if value.Kind() != reflect.Pointer || value.IsNil() || value.Elem().Kind() != reflect.Struct {
		return ctx.ShouldBind(req)
	}
	value = value.Elem()
	trusted := make(map[string]reflect.Value)
	for _, name := range authoritativeFields {
		field := value.FieldByName(name)
		if field.IsValid() && field.CanSet() && !field.IsZero() {
			snapshot := reflect.New(field.Type()).Elem()
			snapshot.Set(field)
			trusted[name] = snapshot
		}
	}
	err := ctx.ShouldBind(req)
	for name, snapshot := range trusted {
		value.FieldByName(name).Set(snapshot)
	}
	if err != nil {
		var validationErrors validator.ValidationErrors
		if !errors.As(err, &validationErrors) {
			return err
		}
	}
	// Validate the final authoritative request, not attacker-supplied route IDs.
	return binding.Validator.ValidateStruct(req)
}

package broker

import (
	"bytes"
	"encoding/json"
	"log"

	"github.com/cloudevents/sdk-go/pkg/cloudevents"
	"github.com/golang/protobuf/jsonpb"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	eventingv1alpha1 "github.com/knative/eventing/pkg/apis/eventing/v1alpha1"
)

func filterEventByCEL(ts *eventingv1alpha1.TriggerSpec, event *cloudevents.Event) bool {
	e, err := cel.NewEnv(
		cel.Declarations(
			decls.NewIdent("ce", decls.NewObjectType("google.protobuf.Struct"), nil),
		),
	)
	if err != nil {
		//TODO do something with error
		return false
	}

	p, iss := e.Parse(ts.Filter.CEL.Expr)
	if iss != nil && iss.Err() != nil {
		//TODO do something with error
		return false
	}
	c, iss := e.Check(p)
	if iss != nil && iss.Err() != nil {
		//TODO do something with error
		return false
	}

	prg, err := e.Program(c)
	if err != nil {
		//TODO do something with error
		return false
	}

	cloudEvent := &structpb.Struct{}
	eventJSON, err := json.Marshal(event.Context.AsV02())
	if err != nil {
		//TODO do something with error
		return false
	}
	if err := jsonpb.Unmarshal(bytes.NewBuffer(eventJSON), cloudEvent); err != nil {
		log.Fatalf("json parse error: %s\n", err)
	}

	out, _, err := prg.Eval(map[string]interface{}{
		"ce": cloudEvent,
	})
	if err != nil {
		//TODO do something with error
		return false
	}

	return out == types.True
}

package broker

import (
	"bytes"
	"encoding/json"

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
			decls.NewIdent("data", decls.NewObjectType("google.protobuf.Struct"), nil),
		),
	)
	if err != nil {
		//TODO do something with error
		return false
	}

	p, iss := e.Parse(ts.Filter.CEL.Expression)
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

	vars := map[string]interface{}{}

	// Create a Struct containing all the known CloudEvents fields. This is the
	// filtering baseline.
	// TODO refactor this so the extra struct isn't allocated if it isn't used
	vars["ce"] = ceContextToStruct(event.Context)

	if ts.Filter.CEL.ParseExtensions {
		func() {
			eventContextStruct := &structpb.Struct{}
			eventContextJSON, err := json.Marshal(event.Context.AsV02())
			if err != nil {
				//TODO do something with error
				return
			}
			if err := jsonpb.Unmarshal(bytes.NewBuffer(eventContextJSON), eventContextStruct); err != nil {
				//TODO do something with error
				return
			}
			// If we get here, replace the static context with the dynamic one
			vars["ce"] = eventContextStruct
		}()
	}

	if ts.Filter.CEL.ParseData {
		func() {
			eventDataStruct := &structpb.Struct{}
			// CloudEvents SDK might have a better way to do this with data codecs
			if event.Context.AsV02().GetDataContentType() == "application/json" {
				eventDataJSON, err := json.Marshal(event.Data)
				if err != nil {
					//TODO do something with error
					return
				}
				if err := jsonpb.Unmarshal(bytes.NewBuffer(eventDataJSON), eventDataStruct); err != nil {
					//TODO do something with error
					return
				}
			}
			vars["data"] = eventDataStruct
		}()
	}

	out, _, err := prg.Eval(vars)
	if err != nil {
		//TODO do something with error
		return false
	}

	return out == types.True
}

func ceContextToStruct(eventCtx cloudevents.EventContext) *structpb.Struct {
	return &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"specversion": &structpb.Value{
				Kind: &structpb.Value_StringValue{
					StringValue: eventCtx.GetSpecVersion(),
				},
			},
			"type": &structpb.Value{
				Kind: &structpb.Value_StringValue{
					StringValue: eventCtx.GetType(),
				},
			},
			"source": &structpb.Value{
				Kind: &structpb.Value_StringValue{
					StringValue: eventCtx.GetSource(),
				},
			},
			"schemaurl": &structpb.Value{
				Kind: &structpb.Value_StringValue{
					StringValue: eventCtx.GetSchemaURL(),
				},
			},
			"datamediatype": &structpb.Value{
				Kind: &structpb.Value_StringValue{
					StringValue: eventCtx.GetDataMediaType(),
				},
			},
			"datacontenttype": &structpb.Value{
				Kind: &structpb.Value_StringValue{
					StringValue: eventCtx.GetDataContentType(),
				},
			},
		},
	}
}

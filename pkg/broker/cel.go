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
	"go.uber.org/zap"
)

const (
	// CELVarKeyContext is the CEL variable key used for the CloudEvent event
	// context.
	CELVarKeyContext = "ce"
	// CELVarKeyData is the CEL variable key used for the CloudEvent event data.
	CELVarKeyData = "data"
)

func (r *Receiver) filterEventByCEL(ts *eventingv1alpha1.TriggerSpec, event *cloudevents.Event) (bool, error) {
	e, err := cel.NewEnv(
		cel.Declarations(
			decls.NewIdent(CELVarKeyContext, decls.NewObjectType("google.protobuf.Struct"), nil),
			decls.NewIdent(CELVarKeyData, decls.NewObjectType("google.protobuf.Struct"), nil),
		),
	)
	if err != nil {
		return false, err
	}

	p, iss := e.Parse(ts.Filter.CEL.Expression)
	if iss != nil && iss.Err() != nil {
		return false, iss.Err()
	}
	c, iss := e.Check(p)
	if iss != nil && iss.Err() != nil {
		return false, iss.Err()
	}

	//TODO cache these by hash of expression. Programs are thread-safe so it's
	// ok to share them between triggers and events.
	prg, err := e.Program(c)
	if err != nil {
		return false, err
	}

	vars := map[string]interface{}{}
	// If the Trigger has requested parsing of extensions, attempt to turn them
	// into a dynamic struct.
	if ts.Filter.CEL.ParseExtensions {
		ctxStruct, err := ceParsedContextStruct(event.Context)
		if err != nil {
			r.logger.Error("Failed to parse event context for CEL filtering", zap.String("id", event.Context.AsV02().ID), zap.Error(err))
		} else {
			vars[CELVarKeyContext] = ctxStruct
		}
	}

	// If the context var wasn't set due to trigger config or a failure to parse
	// extensions, create a struct with the known CE fields as a filtering
	// baseline.
	if _, exists := vars[CELVarKeyContext]; !exists {
		vars[CELVarKeyContext] = ceBaselineContextStruct(event.Context)
	}

	// If the Trigger has requested parsing of data, attempt to turn them into
	// a dynamic struct.
	if ts.Filter.CEL.ParseData {
		dataStruct, err := ceParsedDataStruct(event)
		if err != nil {
			r.logger.Error("Failed to parse event data for CEL filtering", zap.String("id", event.Context.AsV02().ID), zap.Error(err))
		} else {
			vars[CELVarKeyData] = dataStruct
		}
	}

	out, _, err := prg.Eval(vars)
	if err != nil {
		return false, err
	}

	return out == types.True, nil
}

func ceParsedContextStruct(eventCtx cloudevents.EventContext) (*structpb.Struct, error) {
	ctxStruct := &structpb.Struct{}
	//TODO should this coerce to V02?
	ctxJSON, err := json.Marshal(eventCtx.AsV02())
	if err != nil {
		return nil, err
	}
	if err := jsonpb.Unmarshal(bytes.NewBuffer(ctxJSON), ctxStruct); err != nil {
		return nil, err
	}
	return ctxStruct, nil
}

func ceBaselineContextStruct(eventCtx cloudevents.EventContext) *structpb.Struct {
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

func ceParsedDataStruct(event *cloudevents.Event) (*structpb.Struct, error) {
	//TODO CloudEvents SDK might have a better way to do this with data codecs
	if event.Context.GetDataContentType() == "application/json" {
		dataStruct := &structpb.Struct{}
		dataJSON, err := json.Marshal(event.Data)
		if err != nil {
			return nil, err
		}
		if err := jsonpb.Unmarshal(bytes.NewBuffer(dataJSON), dataStruct); err != nil {
			return nil, err
		}
		return dataStruct, nil
	}
	return nil, nil
}

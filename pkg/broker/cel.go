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
	// CELVarKeyExtensions is the CEL variable key used for the CloudEvent event
	// context extensions.
	CELVarKeyExtensions = "ext"
	// CELVarKeyData is the CEL variable key used for the CloudEvent event data.
	CELVarKeyData = "data"
	//TODO add a key that contains both the extensions and the baseline context
	// so extensions can be future proofed
)

func (r *Receiver) filterEventByCEL(ts *eventingv1alpha1.TriggerSpec, event *cloudevents.Event) (bool, error) {
	e, err := cel.NewEnv(
		cel.Declarations(
			decls.NewIdent("specversion", decls.String, nil),
			decls.NewIdent("typ", decls.String, nil),
			decls.NewIdent("source", decls.String, nil),
			decls.NewIdent("schemaurl", decls.String, nil),
			decls.NewIdent("datamediatype", decls.String, nil),
			decls.NewIdent("datacontenttype", decls.String, nil),
			decls.NewIdent(CELVarKeyExtensions, decls.NewObjectType("google.protobuf.Struct"), nil),
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
	// Set baseline context fields
	vars["specversion"] = event.Context.GetSpecVersion()
	// TODO this doesn't work because `type` is reserved in CEL (it's a cast)
	vars["typ"] = event.Context.GetType()
	vars["source"] = event.Context.GetSource()
	vars["schemaurl"] = event.Context.GetSchemaURL()
	vars["datamediatype"] = event.Context.GetDataMediaType()
	vars["datacontenttype"] = event.Context.GetDataContentType()

	// If the Trigger has requested parsing of extensions, attempt to turn them
	// into a dynamic struct.
	if ts.Filter.CEL.ParseExtensions {
		//TODO should this coerce to V02?
		extStruct, err := ceParsedExtensionsStruct(event.Context.AsV02().Extensions)
		if err != nil {
			r.logger.Error("Failed to parse event context for CEL filtering", zap.String("id", event.Context.AsV02().ID), zap.Error(err))
		} else {
			vars[CELVarKeyExtensions] = extStruct
		}
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

func ceParsedExtensionsStruct(ext map[string]interface{}) (*structpb.Struct, error) {
	extJSON, err := json.Marshal(ext)
	if err != nil {
		return nil, err
	}

	extStruct := &structpb.Struct{}
	if err := jsonpb.Unmarshal(bytes.NewBuffer(extJSON), extStruct); err != nil {
		return nil, err
	}
	return extStruct, nil
}

func ceParsedDataStruct(event *cloudevents.Event) (*structpb.Struct, error) {
	//TODO CloudEvents SDK might have a better way to do this with data codecs
	if event.Context.GetDataContentType() == "application/json" {
		var decodedData map[string]interface{}
		err := event.DataAs(&decodedData)
		if err != nil {
			return nil, err
		}
		dataJSON, err := json.Marshal(decodedData)
		if err != nil {
			return nil, err
		}

		dataStruct := &structpb.Struct{}
		//TODO is there a way to convert a map into a structpb.Struct?
		if err := jsonpb.Unmarshal(bytes.NewBuffer(dataJSON), dataStruct); err != nil {
			return nil, err
		}
		return dataStruct, nil
	}
	return nil, nil
}

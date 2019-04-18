package broker

import (
	"github.com/cloudevents/sdk-go"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	eventingv1alpha1 "github.com/knative/eventing/pkg/apis/eventing/v1alpha1"
	"go.uber.org/zap"
)

const (
	// CELVarKeyContext is the CEL variable key used for CloudEvents event
	// context attributes, both official and extension.
	CELVarKeyContext = "ce"
	// CELVarKeyData is the CEL variable key used for parsed, structured event
	// data.
	CELVarKeyData = "data"
)

func (r *Receiver) filterEventByCEL(ts *eventingv1alpha1.TriggerSpec, event *cloudevents.Event) (bool, error) {
	e, err := cel.NewEnv(
		cel.Declarations(
			decls.NewIdent(CELVarKeyContext, decls.Dyn, nil),
			decls.NewIdent(CELVarKeyData, decls.Dyn, nil),
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

	// TODO cache these by hash of expression. Programs are thread-safe so it's
	// ok to share them between triggers and events.
	prg, err := e.Program(c)
	if err != nil {
		return false, err
	}

	// Set baseline context attributes. The attributes available may not be
	// exactly the same as the attributes defined in the current version of the
	// CloudEvents spec.
	ce := map[string]interface{}{
		"specversion": event.SpecVersion(),
		"type":        event.Type(),
		"source":      event.Source(),
		"subject":     event.Subject(),
		"id":          event.ID(),
		// TODO Time. Should this be a string or a (cel-native) protobuf timestamp?
		"schemaurl":           event.SchemaURL(),
		"datacontenttype":     event.DataContentType(),
		"datamediatype":       event.DataMediaType(),
		"datacontentencoding": event.DataContentEncoding(),
	}

	// If the Trigger has requested parsing of extensions, attempt to turn them
	// into a dynamic struct.
	if ts.Filter.CEL.ParseExtensions {
		// TODO should this coerce to V02?
		ext := event.Context.AsV02().Extensions
		if ext != nil {
			for k, v := range ext {
				ce[k] = v
			}
		}
	}

	// If the Trigger has requested parsing of data, attempt to turn them into
	// a dynamic struct.
	data := make(map[string]interface{})
	if ts.Filter.CEL.ParseData {
		data, err = ceParsedData(event)
		if err != nil {
			r.logger.Error("Failed to parse event data for CEL filtering", zap.String("id", event.Context.AsV02().ID), zap.Error(err))
		}
	}

	out, _, err := prg.Eval(map[string]interface{}{
		CELVarKeyContext: ce,
		CELVarKeyData:    data,
	})
	if err != nil {
		return false, err
	}

	return out == types.True, nil
}

func ceParsedData(event *cloudevents.Event) (map[string]interface{}, error) {
	// TODO CloudEvents SDK might have a better way to do this with data codecs
	if event.DataMediaType() == "application/json" {
		var decodedData map[string]interface{}
		err := event.DataAs(&decodedData)
		if err != nil {
			return nil, err
		}
		return decodedData, nil
	}
	return nil, nil
}

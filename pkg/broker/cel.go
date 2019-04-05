package broker

import (
	"github.com/cloudevents/sdk-go/pkg/cloudevents"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	eventingv1alpha1 "github.com/knative/eventing/pkg/apis/eventing/v1alpha1"
	celtypes "github.com/knative/eventing/pkg/broker/dev_knative"
)

func filterEventByCEL(ts *eventingv1alpha1.TriggerSpec, event *cloudevents.Event) bool {
	e, err := cel.NewEnv(
		cel.Types(&celtypes.CloudEventFilterMeta{}),
		cel.Declarations(
			decls.NewIdent("ce", decls.NewObjectType("dev.knative.CloudEventFilterMeta"), nil),
			//decls.NewIdent("data", types.DynType, nil),
		),
	)
	if err != nil {
		//TODO do something with error
		return false
	}

	p, iss := e.Parse(*ts.Filter.CELExpression)
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

	// Would it be useful to cache programs by trigger UID and resourceversion?

	//TODO generate the variables
	ce := &celtypes.CloudEventFilterMeta{
		Specversion: event.SpecVersion(),
		Type:        event.Type(),
		Source:      event.Source(),
		//TODO Is this the right way to get id? Do we even need id?
		//Id: event.Context.AsV02().ID
		//TODO should this use google.protobuf.Timestamp? Do we even need time?
		Time: event.Context.AsV02().Time.String(),
	}

	out, _, err := prg.Eval(map[string]interface{}{
		// Native values are converted to CEL values under the covers.
		"ce": ce,
		//"data": data,
	})
	if err != nil {
		//TODO do something with error
		return false
	}

	return out == types.True
}

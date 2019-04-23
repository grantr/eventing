/*
 * Copyright 2019 The Knative Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
	expr := ts.Filter.CEL.Expression

	prg, err := getOrCacheProgram(expr, programForExpression)
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
		if err := event.DataAs(&data); err != nil {
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

func programForExpression(expr string) (cel.Program, error) {
	env, err := cel.NewEnv(
		cel.Declarations(
			decls.NewIdent(CELVarKeyContext, decls.Dyn, nil),
			decls.NewIdent(CELVarKeyData, decls.Dyn, nil),
		),
	)
	if err != nil {
		return nil, err
	}

	parsed, iss := env.Parse(expr)
	if iss != nil && iss.Err() != nil {
		return nil, iss.Err()
	}
	checked, iss := env.Check(parsed)
	if iss != nil && iss.Err() != nil {
		return nil, iss.Err()
	}

	prg, err := env.Program(checked)
	if err != nil {
		return nil, err
	}
	return prg, nil
}

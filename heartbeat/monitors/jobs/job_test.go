// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package jobs

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/heartbeat/eventext"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/mapval"
	"github.com/elastic/beats/libbeat/testing/mapvaltest"
)

func TestWrapAll(t *testing.T) {
	type args struct {
		jobs []Job
		fns  []JobWrapper
	}

	var basicJob Job = func(event *beat.Event) (jobs []Job, err error) {
		eventext.MergeEventFields(event, common.MapStr{"basic": "job"})
		return nil, nil
	}

	var contJob Job = func(event *beat.Event) (js []Job, e error) {
		eventext.MergeEventFields(event, common.MapStr{"cont": "job"})
		return []Job{basicJob}, nil
	}

	addFoo := func(job Job) Job {
		return func(event *beat.Event) ([]Job, error) {
			cont, err := job(event)
			eventext.MergeEventFields(event, common.MapStr{"foo": "bar"})
			return cont, err
		}
	}

	addBaz := func(job Job) Job {
		return func(event *beat.Event) ([]Job, error) {
			cont, err := job(event)
			eventext.MergeEventFields(event, common.MapStr{"baz": "bot"})
			return cont, err
		}
	}

	tests := []struct {
		name         string
		args         args
		resultFields []mapval.Map
	}{
		{
			"simple",
			args{
				[]Job{basicJob},
				[]JobWrapper{addFoo},
			},
			[]mapval.Map{{"basic": "job", "foo": "bar"}},
		},
		{
			"multijob",
			args{
				[]Job{basicJob, basicJob},
				[]JobWrapper{addFoo},
			},
			[]mapval.Map{
				{"basic": "job", "foo": "bar"},
				{"basic": "job", "foo": "bar"},
			},
		},
		{
			"continuations",
			args{
				[]Job{contJob},
				[]JobWrapper{addFoo},
			},
			[]mapval.Map{
				{"cont": "job", "foo": "bar"},
				{"basic": "job", "foo": "bar"},
			},
		},
		{
			"continuations multi-wrap",
			args{
				[]Job{contJob},
				[]JobWrapper{addFoo, addBaz},
			},
			[]mapval.Map{
				{"cont": "job", "foo": "bar", "baz": "bot"},
				{"basic": "job", "foo": "bar", "baz": "bot"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := ExecJobsAndConts(t, WrapAll(tt.args.jobs, tt.args.fns...))
			require.NoError(t, err)

			for idx, rf := range tt.resultFields {
				fr := results[idx].Fields

				validator := mapval.Strict(mapval.MustCompile(rf))
				mapvaltest.Test(t, validator, fr)
			}
		})
	}
}

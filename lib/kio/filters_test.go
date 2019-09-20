// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kio_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	. "lib.kpt.dev/kio"
)

var r = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo1
  namespace: bar
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo2
---
apiVersion: v1
kind: Service
metadata:
  name: foo2
  namespace: bar
---
apiVersion: v1
kind: Service
metadata:
  name: foo1
`

func TestFileSetter_Filter(t *testing.T) {
	in := bytes.NewBufferString(r)
	out := &bytes.Buffer{}
	err := Pipeline{
		Inputs:  []Reader{ByteReader{Reader: in}},
		Filters: []Filter{&FileSetter{}},
		Outputs: []Writer{ByteWriter{Writer: out}},
	}.Execute()
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, `apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo1
  namespace: bar
  annotations:
    kpt.dev/kio/path: foo1_deployment.yaml
    kpt.dev/kio/mode: 384
---
apiVersion: v1
kind: Service
metadata:
  name: foo1
  annotations:
    kpt.dev/kio/path: foo1_service.yaml
    kpt.dev/kio/mode: 384
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo2
  annotations:
    kpt.dev/kio/path: foo2_deployment.yaml
    kpt.dev/kio/mode: 384
---
apiVersion: v1
kind: Service
metadata:
  name: foo2
  namespace: bar
  annotations:
    kpt.dev/kio/path: foo2_service.yaml
    kpt.dev/kio/mode: 384
`, out.String())
}

func TestFileSetter_Filter_pattern(t *testing.T) {
	in := bytes.NewBufferString(r)
	out := &bytes.Buffer{}
	err := Pipeline{
		Inputs: []Reader{ByteReader{Reader: in}},
		Filters: []Filter{&FileSetter{
			FilenamePattern: "%n_%s_%k.yaml",
		}},
		Outputs: []Writer{ByteWriter{Writer: out}},
	}.Execute()
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, `apiVersion: v1
kind: Service
metadata:
  name: foo1
  annotations:
    kpt.dev/kio/path: foo1__service.yaml
    kpt.dev/kio/mode: 384
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo1
  namespace: bar
  annotations:
    kpt.dev/kio/path: foo1_bar_deployment.yaml
    kpt.dev/kio/mode: 384
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo2
  annotations:
    kpt.dev/kio/path: foo2__deployment.yaml
    kpt.dev/kio/mode: 384
---
apiVersion: v1
kind: Service
metadata:
  name: foo2
  namespace: bar
  annotations:
    kpt.dev/kio/path: foo2_bar_service.yaml
    kpt.dev/kio/mode: 384
`, out.String())
}

func TestFileSetter_Filter_empty(t *testing.T) {
	in := bytes.NewBufferString(r)
	out := &bytes.Buffer{}
	err := Pipeline{
		Inputs: []Reader{ByteReader{Reader: in}},
		Filters: []Filter{&FileSetter{
			FilenamePattern: "resource.yaml",
		}},
		Outputs: []Writer{ByteWriter{Writer: out}},
	}.Execute()
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, `apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo1
  namespace: bar
  annotations:
    kpt.dev/kio/path: resource.yaml
    kpt.dev/kio/mode: 384
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo2
  annotations:
    kpt.dev/kio/path: resource.yaml
    kpt.dev/kio/mode: 384
---
apiVersion: v1
kind: Service
metadata:
  name: foo2
  namespace: bar
  annotations:
    kpt.dev/kio/path: resource.yaml
    kpt.dev/kio/mode: 384
---
apiVersion: v1
kind: Service
metadata:
  name: foo1
  annotations:
    kpt.dev/kio/path: resource.yaml
    kpt.dev/kio/mode: 384
`, out.String())
}
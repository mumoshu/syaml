package syaml

import (
	"bytes"
	"errors"
	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
	"io"
	"testing"
)

func TestHelmToArgoCD(t *testing.T) {
	input := `# Source: myapp/templates/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: RELEASE-NAME-myapp
  namespace: default
  labels:
    app: RELEASE-NAME-myapp
    chart: "myapp-1.6.2"
    release: "RELEASE-NAME"
    heritage: "Helm"
spec:
  selector:
    matchLabels:
      app: RELEASE-NAME-myapp
      release: RELEASE-NAME
  template:
    metadata:
      labels:
        app: RELEASE-NAME-myapp
        release: RELEASE-NAME
    spec:
      containers:
      - name: RELEASE-NAME-myapp
        image: "myapp:5.7.28"
---
# Source: myapp/templates/db-migration.yaml
apiVersion: v1
kind: Pod
metadata:
  name: RELEASE-NAME-myapp-db-migration
  namespace: default
  labels:
    app: RELEASE-NAME-myapp
    chart: "myapp-1.6.2"
    heritage: "Helm"
    release: "RELEASE-NAME"
  annotations:
    "helm.sh/hook": pre-install,pre-upgrade
spec:
  containers:
    - name: RELEASE-NAME-test
      image: "myapp:5.7.28"
      command: ["/tools/db-migrate"]
`

	want := `# Source: myapp/templates/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: RELEASE-NAME-myapp
  namespace: default
  labels:
    app: RELEASE-NAME-myapp
    chart: "myapp-1.6.2"
    release: "RELEASE-NAME"
    heritage: "Helm"
spec:
  selector:
    matchLabels:
      app: RELEASE-NAME-myapp
      release: RELEASE-NAME
  template:
    metadata:
      labels:
        app: RELEASE-NAME-myapp
        release: RELEASE-NAME
    spec:
      containers:
      - name: RELEASE-NAME-myapp
        image: "myapp:5.7.28"
---
# Source: myapp/templates/db-migration.yaml
apiVersion: v1
kind: Pod
metadata:
  name: RELEASE-NAME-myapp-db-migration
  namespace: default
  labels:
    app: RELEASE-NAME-myapp
    chart: "myapp-1.6.2"
    heritage: "Helm"
    release: "RELEASE-NAME"
  annotations:
    "helm.sh/hook": pre-install,pre-upgrade
    "argocd.argoproj.io/hook": "PreSync"
spec:
  containers:
  - name: RELEASE-NAME-test
    image: "myapp:5.7.28"
    command: ["/tools/db-migrate"]
`

	var docs []*yaml.Node

	dec := yaml.NewDecoder(bytes.NewReader([]byte(input)))

	for i := 0; ; i++ {
		var doc yaml.Node

		if err := dec.Decode(&doc); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			t.Fatalf("decoding doc %d: %v", i, err)
		}

		docs = append(docs, &doc)
	}

	patch := Set([]string{"metadata", "annotations", "argocd.argoproj.io/hook"}, "PreSync")

	cond := Eq([]string{"metadata", "annotations", "helm.sh/hook"}, "pre-install,pre-upgrade")

	err1 := Apply(docs[0], patch, cond)
	if !errors.Is(err1, &ValueNotFoundError{}) {
		t.Errorf("The first document should not have matched: %v", err1)
	}

	err := Apply(docs[1], patch, cond)
	if !errors.Is(err, &ValueNotFoundError{}) {
		t.Errorf("The first document should not have matched: %v", err)
	}

	buf := bytes.Buffer{}

	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)

	for i, doc := range docs {
		if err := enc.Encode(doc); err != nil {
			t.Fatalf("encoding doc %d: %v", i, err)
		}
	}

	got := buf.String()

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpcted result: %s", diff)
	}
}

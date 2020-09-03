package integrationtests

const manifestTemplate = `
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: metrics-reader
rules:
- nonResourceURLs: ["/metrics"]
  verbs: ["get"]

---

apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: {{ .ServiceAccountName }}-can-read-metrics
  namespace: {{ .Namespace }}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: metrics-reader
subjects:
- kind: ServiceAccount
  name: {{ .ServiceAccountName }}
  namespace: {{ .Namespace }}

---

apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{ .ServiceAccountName }}
  namespace: {{ .Namespace }}

---

kind: DaemonSet
apiVersion: apps/v1
metadata:
  name: kubelet-proxy-ds
  namespace: {{ .Namespace }}
spec:
  selector:
    matchLabels:
      app: kube-api-proxy
  template:
    metadata:
      labels:
        app: kube-api-proxy
    spec:
      serviceAccountName: {{ .ServiceAccountName }}
      containers:
      - name: kube-api-proxy
        image: kubelet-proxy-dev:latest
        imagePullPolicy: Never
        command:
        - /bin/kubelet-proxy
        args:
        - --paths=/metrics
`

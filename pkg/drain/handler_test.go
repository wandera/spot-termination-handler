package drain

import (
	"reflect"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/kubectl/pkg/drain"
)

func TestHandler_Drain(t *testing.T) {
	type fields struct {
		Client              kubernetes.Interface
		Logger              *zap.Logger
		PodName             string
		Force               bool
		GracePeriodSeconds  int
		IgnoreAllDaemonSets bool
		DeleteEmptyDirData  bool
	}
	type args struct {
		node *v1.Node
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "drain empty node",
			fields: fields{
				Client: fake.NewSimpleClientset(&v1.Node{
					ObjectMeta: metav1.ObjectMeta{Name: "test"},
				}),
				Logger: zaptest.NewLogger(t),
			},
			args: args{
				node: &v1.Node{
					ObjectMeta: metav1.ObjectMeta{Name: "test"},
				},
			},
		},
		{
			name: "drain node with pod",
			fields: fields{
				Client: fake.NewSimpleClientset(
					&v1.Node{
						ObjectMeta: metav1.ObjectMeta{Name: "test"},
					},
					&v1.Pod{
						ObjectMeta: metav1.ObjectMeta{Name: "testpod"},
						Spec:       v1.PodSpec{NodeName: "test"},
					}),
				Logger: zaptest.NewLogger(t),
			},
			args: args{
				node: &v1.Node{
					ObjectMeta: metav1.ObjectMeta{Name: "test"},
				},
			},
		},
		{
			name: "drain node with pod with force",
			fields: fields{
				Client: fake.NewSimpleClientset(
					&v1.Node{
						ObjectMeta: metav1.ObjectMeta{Name: "test"},
					},
					&v1.Pod{
						ObjectMeta: metav1.ObjectMeta{Name: "testpod"},
						Spec:       v1.PodSpec{NodeName: "test"},
					}),
				Logger: zaptest.NewLogger(t),
				Force:  true,
			},
			args: args{
				node: &v1.Node{
					ObjectMeta: metav1.ObjectMeta{Name: "test"},
				},
			},
		},
		{
			name: "nil node",
			fields: fields{
				Client: fake.NewSimpleClientset(),
				Logger: zaptest.NewLogger(t),
			},
			args: args{
				node: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				Client:              tt.fields.Client,
				Logger:              tt.fields.Logger,
				PodName:             tt.fields.PodName,
				Force:               tt.fields.Force,
				GracePeriodSeconds:  tt.fields.GracePeriodSeconds,
				IgnoreAllDaemonSets: tt.fields.IgnoreAllDaemonSets,
				DeleteEmptyDirData:  tt.fields.DeleteEmptyDirData,
			}
			h.Drain(tt.args.node)
		})
	}
}

func Test_buildPodEvent(t *testing.T) {
	type args struct {
		pod               *v1.Pod
		reportingInstance string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "regular event",
			args: args{
				pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "Foo",
						Namespace: "Bar",
					},
				},
				reportingInstance: "instance",
			},
		},
		{
			name: "empty reporting instance",
			args: args{
				pod: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "Foo",
						Namespace: "Bar",
					},
				},
				reportingInstance: "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPodEvent(tt.args.pod, tt.args.reportingInstance)
			if got.GenerateName != tt.args.pod.Name {
				t.Errorf("Pod event GenerateName = %v, want %v", got.GenerateName, tt.args.pod.Name)
			}
			if got.Namespace != tt.args.pod.Namespace {
				t.Errorf("Pod event Namespace = %v, want %v", got.Namespace, tt.args.pod.Namespace)
			}
			if got.ReportingInstance != tt.args.reportingInstance {
				t.Errorf("Pod event ReportingInstance = %v, want %v", got.ReportingInstance, tt.args.reportingInstance)
			}
			if got.Reason != reason {
				t.Errorf("Pod event Reason = %v, want %v", got.ReportingInstance, reason)
			}
			if got.Type != v1.EventTypeNormal {
				t.Errorf("Pod event Type = %v, want %v", got.ReportingInstance, v1.EventTypeNormal)
			}
		})
	}
}

func Test_filterPod(t *testing.T) {
	type args struct {
		podName string
		pod     v1.Pod
	}
	tests := []struct {
		name string
		args args
		want drain.PodDeleteStatus
	}{
		{
			name: "regular pod",
			args: args{
				podName: "some",
				pod: v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "other",
					},
				},
			},
			want: drain.MakePodDeleteStatusOkay(),
		},
		{
			name: "self pod",
			args: args{
				podName: "some",
				pod: v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "some",
					},
				},
			},
			want: drain.MakePodDeleteStatusSkip(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filterPod(tt.args.podName)(tt.args.pod); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filterPod() = %v, want %v", got, tt.want)
			}
		})
	}
}

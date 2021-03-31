package drain

import (
	"context"
	"fmt"
	"spot-termination-handler/pkg/logs"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	v1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/drain"
)

const (
	action              = "delete pod"
	reason              = "SpotTermination"
	reportingController = "wandera/spot-termination-handler"
)

type Handler struct {
	Client              kubernetes.Interface
	Logger              *zap.Logger
	PodName             string
	Force               bool
	GracePeriodSeconds  int
	IgnoreAllDaemonSets bool
	DeleteEmptyDirData  bool
}

func (h *Handler) Drain(node *v1.Node) {
	ctx := context.Background()
	log := h.Logger.Named("drain").Sugar()
	dh := &drain.Helper{
		Ctx:                 ctx,
		Client:              h.Client,
		Force:               h.Force,
		GracePeriodSeconds:  h.GracePeriodSeconds,
		IgnoreAllDaemonSets: h.IgnoreAllDaemonSets,
		DeleteEmptyDirData:  h.DeleteEmptyDirData,
		AdditionalFilters:   []drain.PodFilter{filterPod(h.PodName)},
		Out:                 logs.NewZapWriter(zapcore.InfoLevel, h.Logger.Named("drain")),
		ErrOut:              logs.NewZapWriter(zapcore.ErrorLevel, h.Logger.Named("drain")),
		OnPodDeletedOrEvicted: func(pod *v1.Pod, usingEviction bool) {
			if er := h.generateSpotInterruptionEvent(ctx, pod); er != nil {
				log.Errorf("failed to generate event for pod %s: %s", pod.Name, er)
			}
			log.Infof("%s in namespace %s, evicted", pod.Name, pod.Namespace)
		},
	}

	log.Info("draining node - spot node is being terminated")
	if err := drain.RunCordonOrUncordon(dh, node, true); err != nil {
		log.Errorf("unable to cordon node %s", err)
	} else {
		if pods, err := dh.GetPodsForDeletion(node.Name); err != nil {
			log.Errorf("unable to list pods %s\n", err)
		} else {
			if err := dh.DeleteOrEvictPods(pods.Pods()); err != nil {
				log.Errorf("failed to evict pods %s", err)
			}
		}
	}
}

func (h *Handler) generateSpotInterruptionEvent(ctx context.Context, pod *v1.Pod) error {
	_, err := h.Client.EventsV1().Events(pod.Namespace).Create(ctx, buildPodEvent(pod, h.PodName), metav1.CreateOptions{})
	return err
}

func buildPodEvent(pod *v1.Pod, reportingInstance string) *eventsv1.Event {
	return &eventsv1.Event{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: pod.Name,
			Namespace:    pod.Namespace,
		},
		EventTime:           metav1.NowMicro(),
		ReportingController: reportingController,
		ReportingInstance:   reportingInstance,
		Action:              action,
		Reason:              reason,
		Regarding: v1.ObjectReference{
			Kind:            pod.Kind,
			Namespace:       pod.Namespace,
			Name:            pod.Name,
			UID:             pod.UID,
			APIVersion:      pod.APIVersion,
			ResourceVersion: pod.ResourceVersion,
		},
		Note: fmt.Sprintf("pod %s in namespace %s evicted due to spot node termination", pod.Name, pod.Namespace),
		Type: v1.EventTypeNormal,
	}
}

func filterPod(podName string) func(pod v1.Pod) drain.PodDeleteStatus {
	return func(pod v1.Pod) drain.PodDeleteStatus {
		if pod.Name == podName {
			return drain.MakePodDeleteStatusSkip()
		}
		return drain.MakePodDeleteStatusOkay()
	}
}

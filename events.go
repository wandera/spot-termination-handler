package main

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	events "k8s.io/api/events/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	action              = "delete pod"
	reason              = "SpotTermination"
	reportingController = "wandera/spot-termination-handler"
)

func generateSpotInterruptionEvent(ctx context.Context, pod *v1.Pod) error {
	_, e := clientSet.EventsV1().Events(pod.Namespace).Create(ctx, buildPodEvent(pod), metav1.CreateOptions{})
	return e
}

func buildPodEvent(pod *v1.Pod) *events.Event {
	return &events.Event{
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

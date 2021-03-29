package main

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	events "k8s.io/api/events/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/drain"
)

const (
	action              = "delete pod"
	reason              = "SpotTerminaiton"
	reportingController = "wandera/spot-termination-handler"
)

func generateSpotInterruptionEvents(ctx context.Context, podID string, c *kubernetes.Clientset, list *drain.PodDeleteList) []error {
	var errors []error
	for _, pod := range list.Pods() {
		event := getEvent(pod, podID)
		if _, e := c.EventsV1().Events(pod.Namespace).Create(ctx, event, metav1.CreateOptions{}); e != nil {
			errors = append(errors, e)
		}
	}
	return errors
}

func getEvent(pod v1.Pod, reportingInstance string) *events.Event {
	event := &events.Event{
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
	return event
}

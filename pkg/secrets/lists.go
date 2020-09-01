package secrets

import (
	corev1 "k8s.io/api/core/v1"
)

type Lists struct {
	Used   corev1.SecretList
	Unused corev1.SecretList
}

func podSecretLists(secrets corev1.SecretList, pods corev1.PodList) Lists {
	lists := Lists{
		Used: corev1.SecretList{
			Items: make([]corev1.Secret, 0),
		},
		Unused: corev1.SecretList{
			Items: make([]corev1.Secret, 0),
		},
	}

	for _, sec := range secrets.Items {
		if secretInPods(sec, pods) {
			lists.Used.Items = append(lists.Used.Items, sec)
		} else {
			lists.Unused.Items = append(lists.Unused.Items, sec)
		}
	}
	return lists
}

func secretInPod(secret corev1.Secret, pod corev1.Pod) bool {
	for _, volume := range pod.Spec.Volumes {
		if volume.Secret != nil && volume.Secret.SecretName == secret.Name {
			return true
		}
	}
	return false
}

func secretInPods(secret corev1.Secret, pods corev1.PodList) bool {
	for _, pod := range pods.Items {
		if secretInPod(secret, pod) {
			return true
		}
	}
	return false
}

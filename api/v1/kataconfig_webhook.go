/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	corev1 "k8s.io/api/core/v1"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

// log is for logging in this package.
var kataconfiglog = logf.Log.WithName("kataconfig-resource")

func (r *KataConfig) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:verbs=create;update;delete,path=/validate-api-kataconfiguration-openshift-io-v1-kataconfig,mutating=false,failurePolicy=fail,groups=api.kataconfiguration.openshift.io,resources=kataconfigs,versions=v1,name=vkataconfig.kb.io

var _ webhook.Validator = &KataConfig{}

func (r *KataConfig) Default() {
	kataconfiglog.Info("default", "name", r.Name)
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *KataConfig) ValidateCreate() error {
	kataconfiglog.Info("validate create", "name", r.Name)

	kubeClient, err := getKubeConfigClient()
	if err != nil {
		return err
	}

	kataConfigList := &KataConfigList{}
	listOpts := []client.ListOption{
		client.InNamespace(corev1.NamespaceAll),
	}

	if err := kubeClient.List(context.TODO(), kataConfigList, listOpts...); err != nil {
		return fmt.Errorf("Failed to list KataConfig custom resources: %v", err)
	}

	if len(kataConfigList.Items) > 0 {
		return fmt.Errorf("KataConfig already exists on the cluster")
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *KataConfig) ValidateUpdate(old runtime.Object) error {
	kataconfiglog.Info("validate update", "name", r.Name)

	if r.Spec.Config.SourceImage != "" {
		return fmt.Errorf("SourceImage declaration for KataConfig is not supported in OpenShift")
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *KataConfig) ValidateDelete() error {
	kataconfiglog.Info("validate delete", "name", r.Name)

	kubeClient, err := getKubeConfigClient()
	if err != nil {
		return err
	}

	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(corev1.NamespaceAll),
	}
	if err := kubeClient.List(context.TODO(), podList, listOpts...); err != nil {
		return fmt.Errorf("Failed to list kata pods: %v", err)
	}

	for _, pod := range podList.Items {
		if pod.Spec.RuntimeClassName != nil {
			if *pod.Spec.RuntimeClassName == "kata" || *pod.Spec.RuntimeClassName == "kata-qemu" ||
				*pod.Spec.RuntimeClassName == "kata-clh" || *pod.Spec.RuntimeClassName == "kata-fc" ||
				*pod.Spec.RuntimeClassName == "kata-qemu-virtiofs" {
				return fmt.Errorf("Existing pods using Kata Runtime found. Please delete the pods manually for before deleting the KataConfig")
			}
		}
	}

	return nil
}

func getKubeConfigClient() (client.Client, error) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = AddToScheme(scheme)

	kubeconfig := ctrl.GetConfigOrDie()
	kubeclient, err := client.New(kubeconfig, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}
	return kubeclient, nil
}

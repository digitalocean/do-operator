package controllers

import (
	"fmt"
	"strconv"

	"github.com/digitalocean/godo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func connectionConfigMapForDB(suffix string, owner client.Object, conn *godo.DatabaseConnection) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: owner.GetNamespace(),
			Name:      owner.GetName() + suffix,
		},
		Data: map[string]string{
			"host":     conn.Host,
			"port":     strconv.Itoa(conn.Port),
			"ssl":      fmt.Sprintf("%v", conn.SSL),
			"database": conn.Database,
		},
	}
}

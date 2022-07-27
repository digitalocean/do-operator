package controllers

import (
	"github.com/digitalocean/godo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func credentialsSecretForDefaultDBUser(owner client.Object, db *godo.Database) *corev1.Secret {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: owner.GetNamespace(),
			Name:      owner.GetName() + "-default-credentials",
		},
		StringData: map[string]string{
			"username": db.Connection.User,
			"password": db.Connection.Password,
			"uri":      db.Connection.URI,
		},
	}

	// We assume connection is non-nil, but private connection could be nil.
	if db.PrivateConnection != nil {
		secret.StringData["private_uri"] = db.PrivateConnection.URI
	}

	return secret
}

func credentialsSecretForDBUser(owner client.Object, user *godo.DatabaseUser) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: owner.GetNamespace(),
			Name:      owner.GetName() + "-credentials",
		},
		StringData: map[string]string{
			"username": user.Name,
			"password": user.Password,
			// TODO(awg): Construct uri and private_uri from DB info.
		},
	}
}

package controllers

import (
	"fmt"
	"github.com/digitalocean/godo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/url"
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

func credentialsSecretForDBUser(db *godo.Database, owner client.Object, user *godo.DatabaseUser) (*corev1.Secret, error) {
	secret := &corev1.Secret{
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
		},
	}
	if db.Connection != nil {
		dbUri, err := url.Parse(db.Connection.URI)
		if err != nil {
			return nil, fmt.Errorf("unable to parse connection uri: %s", err)
		}
		dbUri.User = url.UserPassword(user.Name, user.Password)
		secret.StringData["uri"] = dbUri.String()
	}

	if db.PrivateConnection != nil {
		dbPrivateUri, err := url.Parse(db.PrivateConnection.URI)
		if err != nil {
			return nil, fmt.Errorf("unable to parse private connection uri: %s", err)
		}
		dbPrivateUri.User = url.UserPassword(user.Name, user.Password)
		secret.StringData["private_uri"] = dbPrivateUri.String()
	}

	return secret, nil
}

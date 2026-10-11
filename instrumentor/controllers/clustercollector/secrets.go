package clustercollector

import (
	"context"
	"crypto/sha256"
	"slices"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	k8s "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func destinationsSecretsVersionsHash(ctx context.Context, c client.Client, dests *odigosv1.DestinationList) (string, error) {
	var secret k8s.Secret
	resoureceVersions := []string{}
	for _, dest := range dests.Items {
		if dest.Spec.SecretRef != nil {
			if err := c.Get(ctx, client.ObjectKey{Namespace: dest.Namespace, Name: dest.Spec.SecretRef.Name}, &secret); err != nil {
				return "", err
			}
			resoureceVersions = append(resoureceVersions, secret.ResourceVersion)
		}
	}

	// sort the strings for consistent hash
	slices.Sort(resoureceVersions)
	h := sha256.New()
	for _, version := range resoureceVersions {
		h.Write([]byte(version))
	}

	return string(h.Sum(nil)), nil
}
